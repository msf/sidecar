package sidecar_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/msf/sidecar/sidecar"
)

func TestSenderTimeout(t *testing.T) {
	sender, err := sidecar.NewSender("amqp://localhost:5672")
	require.Nil(t, err, "failed")
	defer sender.Close()

	_, err = sender.Do(sidecar.SenderRequest{Host: "h", ID: "id", Path: "p", Timeout: time.Duration(1) * time.Microsecond})
	require.Error(t, err)
}

func TestSendRecv(t *testing.T) {
	sender, err := sidecar.NewSender("amqp://localhost:5672")
	require.Nil(t, err, "failed")
	defer sender.Close()

	recver, err := sidecar.NewReceiver("amqp://localhost:5672", "queue", "http://localhost:81")
	require.Nil(t, err, "failed")
	defer recver.Close()
	go recver.ServeRequestsLoop()

	resp, err := sender.Do(sidecar.SenderRequest{Host: "queue", ID: "id", Path: "/"})
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.Equal(t, 599, resp.StatusCode, "localhost:8080 can't exit")
}

func TestSenderRequestBasic(t *testing.T) {
	req := sidecar.SenderRequest{}
	require.Error(t, req.HasError())

	req.Host = "h"
	req.ID = "id"
	req.Path = "p"
	require.Nil(t, req.HasError())
}
