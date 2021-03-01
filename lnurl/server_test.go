package lnurl

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"lnurl-grpc-proxy/api"
	"net"
	"sync"
	"testing"
	"time"
)

func TestGrpcServer_LnurlWithdraw(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	lis := bufconn.Listen(1024 * 1024)
	defer lis.Close()

	srv := grpc.NewServer()

	withdrawer := &WithdrawerMock{}
	wps := NewGrpcServer(withdrawer)
	defer wps.Stop()

	api.RegisterWithdrawProxyServer(srv, wps)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		err := srv.Serve(lis)
		assert.NoError(t, err, "expected no error")
		wg.Done()
	}()

	cc, err := grpc.Dial("bufconn", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("expected no error: %v", err)
	}

	wpc := api.NewWithdrawProxyClient(cc)
	wpgrpc, err := wpc.LnurlWithdraw(context.Background())
	if err != nil {
		t.Fatalf("expected no error: %v", err)
	}

	withdrawer.On("AddWithdrawRequest", mock.Anything, mock.Anything, mock.Anything).Return("123", nil)

	err = wpgrpc.Send(&api.LnurlWithdrawRequest{
		Event: &api.LnurlWithdrawRequest_Open{
			Open: &api.OpenWithdraw{
				WithdrawId:  "myid",
				MinAmount:   123,
				MaxAmount:   500,
				Description: "mydesc",
			},
		},
	})
	assert.NoError(t, err, "expected no error")

waitForCall:
	for {
		select {
		case <-ctx.Done():
			t.Errorf("context done before results checked in")
		default:
			if len(withdrawer.Calls) >= 1 {
				break waitForCall
			}
		}
	}

	srv.Stop()
	wg.Wait()

	withdrawer.AssertExpectations(t)
}
