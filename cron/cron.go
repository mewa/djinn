package cron

type Dcron struct{}

func New() *Dcron {
	return &Dcron{}
}

func (c *Dcron) PutEntry(entry Entry) {}

func (c *Dcron) DeleteEntry(id EntryID) {}

func (c *Dcron) Entry(id EntryID) Entry {
	return Entry{}
}

func (c *Dcron) Start() {}

func (c *Dcron) Stop() {}
