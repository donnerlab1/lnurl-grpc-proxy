package lnurl

import (
	"context"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"lnurl-grpc-proxy/api"
	"log"
)

var (
	unkownError = status.Error(codes.Unknown, "something went wrong")
)

type GrpcServer struct {
	withdrawer Withdrawer

	ctx context.Context
	cancel context.CancelFunc
}


func NewGrpcServer(withdrawer Withdrawer) *GrpcServer {
	ctx, cancel := context.WithCancel(context.Background())
	return &GrpcServer{withdrawer: withdrawer, ctx: ctx, cancel: cancel}
}

func (g *GrpcServer) LnurlWithdraw(server api.WithdrawProxy_LnurlWithdrawServer) error {

	lnurlClient := &GrpcWithdrawClient{
		invoiceChan: make(chan string),
		errChan:     make(chan error),
	}
	defer lnurlClient.Close()
	msg, err := server.Recv()
	if err != nil {
		return status.Errorf(codes.Unknown, err.Error())
	}
	openReq := msg.GetOpen()
	if openReq == nil {
		return status.Errorf(codes.Unknown, err.Error())
	}

	log.Printf("\t [GRPC] > New WithdrawReq: %s", openReq.WithdrawId)
	// get bechstring
	bechstring, err := g.withdrawer.AddWithdrawRequest(openReq.WithdrawId, lnurlClient, &WithdrawParams{
		MinAmt:      openReq.MinAmount,
		MaxAmt:      openReq.MaxAmount,
		Description: openReq.Description,
	})
	if err != nil {
		return status.Errorf(codes.Unknown, err.Error())
	}
	// send lnurl-bechstring
	err = server.Send(&api.LnurlWithdrawResponse{Event: &api.LnurlWithdrawResponse_BechString{BechString: &api.LnurlString{BechString: bechstring}}})
	if err != nil {
		return status.Errorf(codes.Unknown, err.Error())
	}

	// Wait for payinvoice request
	var invoice string
Loop:
	for {
		select {
		case <-g.ctx.Done():
			return status.Errorf(codes.Canceled, "context canceled by server")
		case <-server.Context().Done():
			log.Printf("\t [GRPC] > context canceled: %s", openReq.WithdrawId)
			return nil
		case invoice = <-lnurlClient.invoiceChan:
			break Loop
		}
	}

	// send invoice request
	err = server.Send(&api.LnurlWithdrawResponse{Event: &api.LnurlWithdrawResponse_Invoice{Invoice: &api.Invoice{Invoice: invoice}}})
	if err != nil {
		return status.Errorf(codes.Unknown, err.Error())
	}
	// wait for okay
	msg, err = server.Recv()
	if err != nil {
		return status.Errorf(codes.Unknown, err.Error())
	}
	ok := msg.GetPay()
	if ok == nil {
		lnurlClient.errChan <- unkownError
		return unkownError
	}
	if ok.Status == "OK" {
		lnurlClient.errChan <- nil
	} else {
		lnurlClient.errChan <- fmt.Errorf("%s", ok.Reason)
		return fmt.Errorf("%s", ok.Reason)
	}
	return nil
}

func (g *GrpcServer) Stop() {
	g.cancel()
}

type GrpcWithdrawClient struct {
	invoiceChan chan string
	errChan     chan error
}

func (d *GrpcWithdrawClient) PayInvoice(invoice string) error {
	d.invoiceChan <- invoice
	return <-d.errChan
}

func (d *GrpcWithdrawClient) Close() {
	close(d.invoiceChan)
	close(d.errChan)
}
