package djinn

import (
	"context"
	"fmt"
	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/etcdserver/etcdserverpb"
	"github.com/mewa/djinn/utils"
	"go.uber.org/zap"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func srvUrl(srv *net.SRV) string {
	return fmt.Sprintf("%s:%d", strings.TrimSuffix(srv.Target, "."), srv.Port)
}

func (d *Djinn) configure() error {
	var ipHost url.URL
	var srv *net.SRV
	var records []*net.SRV
	var err error

	d.log.Info("resolving", zap.String("name", d.name), zap.String("discovery_dns", d.config.DNSCluster))
	utils.Backoff(100*time.Millisecond, 30*time.Second, func() error {
		ipHost, srv, records, err = d.resolveService("etcd-server")
		return err
	})

	if err != nil {
		return err
	}
	d.log.Info("resolved host", zap.String("name", d.name), zap.String("dns_host", d.host.String()), zap.String("ip_host", ipHost.String()))

	var srvHost url.URL = *d.host
	srvHost.Host = srvUrl(srv)

	d.config.APUrls = []url.URL{*d.host}
	d.config.LPUrls = []url.URL{ipHost}

	// disable client access
	d.config.LCUrls = nil
	d.config.ACUrls = nil

	err = d.updateMembership(records, srvHost)
	if err != nil {
		d.log.Error("could not update membership", zap.String("name", d.name), zap.Error(err))
		return err
	}

	return nil
}

func isNamedMember(d *Djinn, member *etcdserverpb.Member) bool {
	for _, peerUrl := range d.config.APUrls {
		if d.name == member.Name {
			return true
		}
	}
	return false
}

func isPeer(d *Djinn, member *etcdserverpb.Member) bool {
	for _, peerUrl := range d.config.APUrls {
		for _, memberPeerUrl := range member.PeerURLs {
			if peerUrl.String() == memberPeerUrl {
				return true
			}
		}
	}
	return false
}

func (d *Djinn) updateMembership(records []*net.SRV, self url.URL) error {
	endpoints := []string{}
	for _, rec := range records {
		endpoint := srvUrl(rec)
		if endpoint != self.Host {
			endpoints = append(endpoints, endpoint)
		}
	}

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 15 * time.Second,
	})

	if err != nil {
		// cluster doesn't exist yet
		return nil
	}
	defer cli.Close()
	d.log.Info("joining existing cluster", zap.String("name", d.name))

	resp, err := cli.MemberList(context.Background())
	if err != nil {
		d.log.Error("error listing members", zap.Error(err))
		return err
	}

	var added *clientv3.MemberAddResponse
	var named, peer bool
	var me *etcdserverpb.Member
	for _, member := range resp.Members {
		named = isNamedMember(d, member)
		peer = isPeer(d, member)

		if named || peer {
			me = member
			break
		}
	}

	if named {
		// previously present in the cluster
		// this means we're recovering from failure
		_, err := cli.MemberRemove(context.Background(), me.ID)
		if err != nil {
			d.log.Error("error removing old member", zap.String("name", me.Name), zap.Uint64("old_id", me.ID), zap.Error(err))
		} else {
			d.log.Info("removed old member", zap.String("name", me.Name), zap.Uint64("old_id", me.ID))
		}
	}

	if named || !peer {
		// if we were a named member we have just been removed so we need to add ourselves back
		added, err = cli.MemberAdd(context.Background(), []string{self.String()})
		if err != nil {
			d.log.Error("error adding new member", zap.String("name", d.name), zap.String("url", self.String()), zap.Error(err))
			return err
		}

		d.log.Info("added new member", zap.String("name", d.name), zap.Uint64("id", added.Member.ID))

		d.config.ClusterState = "existing"
	}
	// else: if we're not named but are a peer we're creating a
	// new cluster and are in the initial configuration

	return nil
}

func (d *Djinn) resolveService(svc string) (url.URL, *net.SRV, []*net.SRV, error) {
	ips := ipAddresses()

	svcUrl := *d.host
	var mySRV *net.SRV

	_, records, err := net.LookupSRV(svc, "tcp", d.config.DNSCluster)
	if err != nil {
		return url.URL{}, nil, nil, err
	}

Loop:
	for _, record := range records {
		target := strings.TrimSuffix(record.Target, ".")
		svcIPs, _ := net.LookupIP(target)

		for _, hostip := range ips {
			for _, addr := range svcIPs {
				if addr.Equal(hostip) && svcUrl.Port() == strconv.Itoa(int(record.Port)) {
					svcUrl.Host = addr.String() + ":" + svcUrl.Port()

					mySRV = record
					break Loop
				}
			}
		}
	}
	if mySRV == nil {
		return url.URL{}, nil, nil, ErrCannotResolveService
	}

	return svcUrl, mySRV, records, err
}

func ipAddresses() []net.IP {
	addrs, _ := net.InterfaceAddrs()
	ips := []net.IP{}
	for _, addr := range addrs {
		ipaddr, _, err := net.ParseCIDR(addr.String())
		if err != nil {
		}
		ips = append(ips, ipaddr)
	}
	return ips
}
