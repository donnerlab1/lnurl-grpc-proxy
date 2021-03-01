package lnurl

import (
	"fmt"
	"github.com/fiatjaf/go-lnurl"
	"log"
	"strings"
	"sync"
)

const LNURL_WITHDRAWTAG = "withdrawRequest"

var (
	WithdrawNotExistError = fmt.Errorf("withdraw id does not exist")
)

type LnurlWithdrawer interface {
	AddWithdrawRequest(withdrawId string, receiver LnUrlWithdrawReceiver, params *WithdrawParams) (bechstring string, err error)
	WithdrawRequest(withdrawId string) (*lnurl.LNURLWithdrawResponse, *lnurl.LNURLErrorResponse)
	SendInvoice(withdrawId string, invoice string) *lnurl.LNURLErrorResponse
}

type LnUrlWithdrawReceiver interface {
	PayInvoice(invoice string) error
}

type Service struct {
	sync.RWMutex
	baseUrl string

	withdrawMap map[string]*WithdrawProcess
}
type WithdrawProcess struct {
	Receiver       LnUrlWithdrawReceiver
	WithdrawParams *WithdrawParams
}

type WithdrawParams struct {
	MinAmt      int64
	MaxAmt      int64
	Description string
}

func NewService(baseUrl string) *Service {
	srv := &Service{baseUrl: baseUrl}
	srv.withdrawMap = make(map[string]*WithdrawProcess)
	return srv
}

func (s *Service) AddWithdrawRequest(withdrawId string, receiver LnUrlWithdrawReceiver, params *WithdrawParams) (bechstring string, err error) {

	url := fmt.Sprintf("%s/withdraw/%s", s.baseUrl, withdrawId)
	bechstring, err = lnurl.LNURLEncode(url)
	if err != nil {
		return "", err
	}
	process := &WithdrawProcess{
		Receiver:       receiver,
		WithdrawParams: params,
	}

	s.Lock()
	s.withdrawMap[withdrawId] = process
	s.Unlock()

	log.Printf("\t [LNURL] > New WithdrawProcess %s %v", withdrawId, params)
	return bechstring, err
}

func (s *Service) WithdrawRequest(withdrawId string) (*lnurl.LNURLWithdrawResponse, *lnurl.LNURLErrorResponse) {
	s.RLock()

	if withdrawProcess, ok := s.withdrawMap[withdrawId]; ok {
		s.RUnlock()
		res := &lnurl.LNURLWithdrawResponse{
			Tag:                LNURL_WITHDRAWTAG,
			K1:                 withdrawId,
			Callback:           fmt.Sprintf("%s/invoice", s.baseUrl),
			CallbackURL:        nil,
			MaxWithdrawable:    withdrawProcess.WithdrawParams.MaxAmt,
			MinWithdrawable:    withdrawProcess.WithdrawParams.MinAmt,
			DefaultDescription: withdrawProcess.WithdrawParams.Description,
		}
		log.Printf("\t [LNURL] > New WithdrawRequest %s %v", withdrawId, res)
		return res, nil
	}
	s.RUnlock()

	return nil, &lnurl.LNURLErrorResponse{
		Status: "ERROR",
		Reason: WithdrawNotExistError.Error(),
	}
}

func (s *Service) SendInvoice(withdrawId string, invoice string) *lnurl.LNURLErrorResponse {
	s.Lock()
	if process, ok := s.withdrawMap[withdrawId]; ok {
		delete(s.withdrawMap, withdrawId)
		s.Unlock()

		log.Printf("\t [LNURL] > New SendInvoice %s %s", withdrawId, invoice)
		err := process.Receiver.PayInvoice(invoice)
		if err != nil {
			log.Printf("\t [LNURL-ERROR] > Payinvoice %s", withdrawId)
			return &lnurl.LNURLErrorResponse{
				Status: "ERROR",
				Reason: err.Error(),
			}
		}

		log.Printf("\t [LNURL] > SUCCESS Payinvoice %s ", withdrawId)
		return &lnurl.LNURLErrorResponse{
			Status: "OK",
		}
	}
	s.Unlock()
	return &lnurl.LNURLErrorResponse{
		Status: "ERROR",
		Reason: WithdrawNotExistError.Error(),
	}
}

func splitUrl(url string) (withdrawId string) {
	return strings.Split(url, "/")[4]
}
