package lnurl

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"net/http"
)

type RestHandler struct {
	LnurlWithdrawer Withdrawer
}

func NewRestHandler(lnurlWithdrawer Withdrawer) *RestHandler {
	return &RestHandler{LnurlWithdrawer: lnurlWithdrawer}
}

func (rh *RestHandler) GetWithdrawParams(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	id := q.Get("id")
	if id == "" {
		http.Error(w, "empty id", http.StatusBadRequest)
	}

	res, lnurlErr := rh.LnurlWithdrawer.WithdrawRequest(id)

	// return lnurlErr
	if lnurlErr != nil {
		jsErrRes, err := json.Marshal(lnurlErr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(jsErrRes)
		return
	}

	// return lnurlRes
	jsRes, err := json.Marshal(res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(jsRes)
}

func (rh *RestHandler) SendInvoice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()
	withdrawId := query.Get("k1")
	invoice := query.Get("pr")
	res := rh.LnurlWithdrawer.ForwardInvoice(withdrawId, invoice)
	jsRes, err := json.Marshal(res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(jsRes)
}

func (rh *RestHandler) Listen(host string) error {
	router := mux.NewRouter().StrictSlash(true)

	router.HandleFunc("/withdraw/{id}", rh.GetWithdrawParams)
	router.HandleFunc("/invoice", rh.SendInvoice)

	return http.ListenAndServe(host, router)
}
