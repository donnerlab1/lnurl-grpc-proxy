package lnurl

import (
	"fmt"
	"github.com/fiatjaf/go-lnurl"
	"log"
	"net/url"
	"sync"
)

const (
	LNURL_WITHDRAWTAG       = "withdrawRequest"
	DEFAULT_CALLBACK_SUBDIR = "invoice"
	DEFAULT_WITHDRAW_SUBDIR = "withdraw"
)

var (
	WithdrawNotExistError = fmt.Errorf("withdraw id does not exist")
)

type Withdrawer interface {
	AddWithdrawRequest(withdrawId string, receiver InvoicePayer, params *WithdrawParams) (bechstring string, err error)
	WithdrawRequest(withdrawId string) (*lnurl.LNURLWithdrawResponse, *lnurl.LNURLErrorResponse)
	ForwardInvoice(withdrawId string, invoice string) *lnurl.LNURLErrorResponse
}

type InvoicePayer interface {
	PayInvoice(invoice string) error
}

type WithdrawProcess struct {
	Receiver       InvoicePayer
	WithdrawParams *WithdrawParams
}

type WithdrawParams struct {
	MinAmt      int64
	MaxAmt      int64
	Description string
}

type Service struct {
	sync.RWMutex

	baseUrl           string
	withdrawProcesses map[string]*WithdrawProcess
}

func NewService(baseUrl string) *Service {
	srv := &Service{
		baseUrl:           baseUrl,
		withdrawProcesses: make(map[string]*WithdrawProcess),
	}
	return srv
}

// AddWithdrawRequest adds a new withdraw request from a
// LN SERVICE to the withdrawProcess list. This request is
// in further identified by the given id. This returns the
// bechstring that is presented to the LN WALLET as a QR
// code.
func (s *Service) AddWithdrawRequest(id string, receiver InvoicePayer, params *WithdrawParams) (bechstring string, err error) {
	url := fmt.Sprintf("%s/%s?id=%s", s.baseUrl, DEFAULT_WITHDRAW_SUBDIR, id)

	bechstring, err = lnurl.LNURLEncode(url)
	if err != nil {
		return "", err
	}
	process := &WithdrawProcess{
		Receiver:       receiver,
		WithdrawParams: params,
	}

	s.Lock()
	s.withdrawProcesses[id] = process
	s.Unlock()

	log.Printf("\t [LNURL] > New WithdrawProcess %s %v", id, params)
	return bechstring, err
}

// WithdrawRequest handles the request comming from the
// LN WALLET to the LN SERVICE. It returns the Withdraw
// response as specified in the lnurl rfc.
func (s *Service) WithdrawRequest(id string) (*lnurl.LNURLWithdrawResponse, *lnurl.LNURLErrorResponse) {
	// TODO: maybe we should ask the LN SERVICE for the params (max, min withdrawable) in this step.
	s.RLock()

	if withdrawProcess, ok := s.withdrawProcesses[id]; ok {
		s.RUnlock()
		res := &lnurl.LNURLWithdrawResponse{
			Tag:                LNURL_WITHDRAWTAG,
			K1:                 id,
			Callback:           fmt.Sprintf("%s/%s", s.baseUrl, DEFAULT_CALLBACK_SUBDIR),
			CallbackURL:        nil,
			MaxWithdrawable:    withdrawProcess.WithdrawParams.MaxAmt,
			MinWithdrawable:    withdrawProcess.WithdrawParams.MinAmt,
			DefaultDescription: withdrawProcess.WithdrawParams.Description,
		}
		log.Printf("\t [LNURL] > New WithdrawRequest %s %v", id, res)
		return res, nil
	}
	s.RUnlock()

	return nil, &lnurl.LNURLErrorResponse{
		Status: "ERROR",
		Reason: WithdrawNotExistError.Error(),
	}
}

// ForwardInvoice forwards the invoice given by the LN
// WALLET to the LN SERVICE to be payed.
func (s *Service) ForwardInvoice(id string, invoice string) *lnurl.LNURLErrorResponse {
	s.Lock()
	if process, ok := s.withdrawProcesses[id]; ok {
		delete(s.withdrawProcesses, id)
		s.Unlock()

		log.Printf("\t [LNURL] > New ForwardInvoice %s %s", id, invoice)
		err := process.Receiver.PayInvoice(invoice)
		if err != nil {
			log.Printf("\t [LNURL-ERROR] > Payinvoice %s", id)
			return &lnurl.LNURLErrorResponse{
				Status: "ERROR",
				Reason: err.Error(),
			}
		}

		log.Printf("\t [LNURL] > SUCCESS Payinvoice %s ", id)
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

func getIDFromRawUrl(raw string) (id string, err error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}

	id = u.Query().Get("id")
	if id == "" {
		return "", fmt.Errorf("empty id")
	}

	return id, nil
}
