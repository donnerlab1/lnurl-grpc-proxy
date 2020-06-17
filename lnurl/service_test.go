package lnurl

import (
	"fmt"
	"github.com/fiatjaf/go-lnurl"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_Url(t *testing.T) {
	testpk := "4324"
	testid := "4345-53453"
	id := fmt.Sprintf("%s;%s", testpk, testid)
	url := fmt.Sprintf("https://gude/withdraw/%s", id)
	withdrawId := splitUrl(url)
	assert.Equal(t, id, withdrawId)
}
func Test_Service(t *testing.T) {
	lnurlService := NewService("https://gude")
	testClient := &TestClient{
		"gude",
	}

	url, err := lnurlService.AddWithdrawRequest(testClient.withdrawId, testClient, &WithdrawParams{
		MinAmt:      0,
		MaxAmt:      1000,
		Description: "foo",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("lnurl: %s", url)
	decoded, err := lnurl.LNURLDecode(url)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("lnurldecoded: %s", decoded)
	withdrawId := splitUrl(decoded)
	assert.Equal(t, testClient.withdrawId, withdrawId)

	res, errRes := lnurlService.WithdrawRequest(withdrawId)
	if errRes != nil {
		t.Fatal(err)
	}
	assert.Equal(t, res.K1, withdrawId)

	errRes = lnurlService.SendInvoice(withdrawId, "invoice")
	assert.Equal(t, errRes.Status, "OK")
}

type TestClient struct {
	withdrawId string
}

func (t *TestClient) PayInvoice(invoice string) error {
	return nil
}
