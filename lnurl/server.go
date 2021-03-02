package lnurl

import (
	"context"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"github.com/donnerlab1/lnurl-grpc-proxy/api"
	"log"
)

var (
	unkownError = status.Error(codes.Unknown, "something went wrong")
)

type GrpcServer struct {
	withdrawer Withdrawer

	ctx    context.Context
	cancel context.CancelFunc
}

func NewGrpcServer(withdrawer Withdrawer) *GrpcServer {
	ctx, cancel := context.WithCancel(context.Background())
	return &GrpcServer{withdrawer: withdrawer, ctx: ctx, cancel: cancel}
}

func (s *GrpcServer) LnurlWithdraw(server api.WithdrawProxy_LnurlWithdrawServer) error {
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
	bechstring, err := s.withdrawer.AddWithdrawRequest(openReq.WithdrawId, lnurlClient, &WithdrawParams{
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

	for {
		select {
		case <-s.ctx.Done():
			return status.Errorf(codes.Canceled, "canceled by server")
		case <-server.Context().Done():
			log.Printf("\t [GRPC] > context canceled: %s", openReq.WithdrawId)
			return status.Errorf(codes.Canceled, "canceled by the caller")
		case invoice = <-lnurlClient.invoiceChan:
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
			payRes := msg.GetPay()
			if payRes == nil {
				lnurlClient.errChan <- unkownError
				return status.Errorf(codes.Unknown, "payment response was nil")
			}
			if payRes.Status == "OK" {
				lnurlClient.errChan <- nil
			} else {
				lnurlClient.errChan <- fmt.Errorf("%s", payRes.Reason)
			}
			return nil
		}
	}
}

func (s *GrpcServer) Stop() {
	s.cancel()
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
