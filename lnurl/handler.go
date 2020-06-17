package lnurl

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"net/http"
)

type RestHandler struct {
	LnurlWithdrawer LnurlWithdrawer
}

func NewRestHandler(lnurlWithdrawer LnurlWithdrawer) *RestHandler {
	return &RestHandler{LnurlWithdrawer: lnurlWithdrawer}
}

func (rh *RestHandler) GetWithdrawParams(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	withdrawId := vars["id"]

	res, errRes := rh.LnurlWithdrawer.WithdrawRequest(withdrawId)
	if errRes != nil {
		err := json.NewEncoder(w).Encode(errRes)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	}
	err := json.NewEncoder(w).Encode(res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (rh *RestHandler) SendInvoice(w http.ResponseWriter, r *http.Request) {

	query := r.URL.Query()
	withdrawId := query.Get("k1")
	invoice := query.Get("invoice")
	res := rh.LnurlWithdrawer.SendInvoice(withdrawId, invoice)
	err := json.NewEncoder(w).Encode(res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (rh *RestHandler) Listen(host string) error {
	router := mux.NewRouter().StrictSlash(true)

	router.HandleFunc("/withdraw/{id}", rh.GetWithdrawParams)
	router.HandleFunc("/invoice", rh.SendInvoice)

	return http.ListenAndServe(host, router)
}
