package cron

type Dcron struct{}

func New() *Dcron {
	return &Dcron{}
}

func (c *Dcron) AddEntry(entry Entry) {}

func (c *Dcron) UpsertEntry(entry Entry) {}

func (c *Dcron) RemoveEntry(id EntryID) {}

func (c *Dcron) UpdateEntry(entry Entry) {}

func (c *Dcron) Entry(id EntryID) Entry {
	return Entry{}
}

func (c *Dcron) Start() {}

func (c *Dcron) Stop() {}
