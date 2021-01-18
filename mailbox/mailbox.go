package mailbox

import (
	"fmt"
	"sync"
)

// Response carries the equivalent of an http response
type Response struct {
	StatusCode int
	Body       []byte
	Err        error
}

// Mailbox holds a directory of reply channels so that test Responses can return to the goroutine that spowned the test
type Mailbox struct {
	channelMapper *sync.Map
	capacity      int
}

func New() *Mailbox {
	mailbox := &Mailbox{
		channelMapper: &sync.Map{},
		capacity:      1,
	}
	return mailbox
}

func (mb *Mailbox) GetChannel(id string) (chan Response, error) {
	value, ok := mb.channelMapper.Load(id)
	if !ok {
		return nil, fmt.Errorf("There is no channel registered with id:" + id)
	}

	channel, ok := value.(chan Response)
	if !ok {
		return nil, fmt.Errorf("The channel " + id + " had an incorrect type")

	}

	return channel, nil
}

func (mb *Mailbox) Register(id string) (chan Response, error) {
	_, err := mb.GetChannel(id)
	if err == nil {
		return nil, fmt.Errorf("There is already a channel registered with id:" + id)
	}

	channel := make(chan Response, mb.capacity)
	mb.channelMapper.Store(id, channel)
	return channel, nil
}

func (mb *Mailbox) Deregister(id string) {
	mb.channelMapper.Delete(id)
}
