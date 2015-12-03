package general

import (
	"encoding/json"
	"errors"
	"fmt"
	// "strings"
	yf_http "yf_pkg/net/http"
	"yf_pkg/utils"
)

const (
	iapPassword   = "ccac10aa0bf942e7b8acd9dc93a80104"
	iapUrl        = "buy.itunes.apple.com"
	iapSandboxUrl = "sandbox.itunes.apple.com"
)

type Receipt struct {
	Quantity       string `json:"quantity"`
	Product_id     string `json:"product_id"`
	Transaction_id string `json:"transaction_id"`
	Purchase_date  string `json:"purchase_date"`
}

type IapResult struct {
	Status  int `json:"status"`
	Receipt struct {
		In_app []Receipt `json:"in_app"`
	} `json:"receipt"`
}

func IapQuery(receiptdata string, ifsandbox int, transaction string) (count int, Product_id string, transaction_id string, Purchase_date string, e error) {
	// r := &Receipt{1, "a1", "", ""}
	// return r, nil
	content := make(map[string]interface{})
	content["receipt-data"] = receiptdata
	content["password"] = iapPassword
	j, e := json.Marshal(content)
	if e != nil {
		return
	}
	var chost string
	if ifsandbox == 0 {
		chost = iapUrl
	} else {
		chost = iapSandboxUrl
	}
	body, e := yf_http.Send("https", chost, "verifyReceipt", nil, nil, nil, j, 4)
	if e != nil {
		return 0, "", "", "", e
	}
	fmt.Println("verifyReceipt body " + string(body))
	var result IapResult
	if e := json.Unmarshal(body, &result); e != nil {
		return 0, "", "", "", e
	}
	if result.Status != 0 {
		e = errors.New(fmt.Sprintf("%v", result.Status))
		return
	}
	if len(result.Receipt.In_app) <= 0 {
		e = errors.New("no result")
		return
	}
	// fmt.Println("begin check transaction " + transaction)
	if transaction != "" {
		for _, v := range result.Receipt.In_app {
			// fmt.Println("v.Transaction_id " + v.Transaction_id)
			if v.Transaction_id == transaction {

				count, _ = utils.ToInt(v.Quantity)
				Product_id = v.Product_id
				transaction_id = v.Transaction_id
				Purchase_date = v.Purchase_date
			}

		}
		if Product_id == "" {
			e = errors.New("no result")
			return
		}
	} else {
		count, _ = utils.ToInt(result.Receipt.In_app[0].Quantity)
		Product_id = result.Receipt.In_app[0].Product_id
		transaction_id = result.Receipt.In_app[0].Transaction_id
		Purchase_date = result.Receipt.In_app[0].Purchase_date
	}
	return
}
