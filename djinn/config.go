package djinn

import (
	"context"
	"fmt"
	"github.com/coreos/etcd/clientv3"
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
	var host url.URL
	var srv *net.SRV
	var records []*net.SRV
	var err error

	utils.Backoff(100*time.Millisecond, 30*time.Second, func() error {
		host, srv, records, err = d.resolveService("etcd-server", *d.host)
		return err
	})

	if err != nil {
		return err
	}

	var srvHost url.URL = *d.host
	srvHost.Host = srvUrl(srv)

	d.config.APUrls = []url.URL{*d.host}
	d.config.LPUrls = []url.URL{host}

	// disable client access
	d.config.LCUrls = nil
	d.config.ACUrls = nil

	err = d.updateMembership(records, srvHost)
	if err != nil {
		return err
	}

	return nil
}

func (d *Djinn) updateMembership(records []*net.SRV, self url.URL) error {
	endpoints := []string{}
	for _, rec := range records {
		endpoint := fmt.Sprintf("%s:%d", rec.Target, rec.Port)
		endpoints = append(endpoints, endpoint)
	}

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 2 * time.Second,
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
	for _, member := range resp.Members {
		if member.Name == d.name {
			_, err := cli.MemberRemove(context.Background(), member.ID)
			if err != nil {
				d.log.Error("error removing old member", zap.String("name", member.Name), zap.Uint64("old_id", member.ID), zap.Error(err))
				return err
			}

			d.log.Info("removed old member", zap.String("name", member.Name), zap.Uint64("old_id", member.ID))

			added, err = cli.MemberAdd(context.Background(), []string{self.String()})
			if err != nil {
				d.log.Error("error adding new member", zap.String("name", d.name), zap.Error(err))
				return err
			}

			d.log.Info("added new member", zap.String("name", member.Name), zap.Uint64("id", added.Member.ID))
			break
		}
	}

	d.config.ClusterState = "existing"
	return nil
}

func (d *Djinn) resolveService(svc string, u *url.URL) (url.URL, *net.SRV, []*net.SRV, error) {
	ips := ipAddresses()
	svcUrl := *u
	var mySRV *net.SRV

	_, records, err := net.LookupSRV(svc, "tcp", d.config.DNSCluster)
	if err != nil {
		return url.URL{}, nil, nil, err
	}

Loop:
	for _, val := range records {
		svcIPs, _ := net.LookupIP(val.Target)

		for _, hostip := range ips {
			for _, addr := range svcIPs {
				if addr.Equal(hostip) {
					svcUrl.Host = addr.String() + ":" + svcUrl.Port()
					mySRV = val
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
