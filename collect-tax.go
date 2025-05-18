package main

import (
	"encoding/json"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
	shell "github.com/stateless-minds/go-ipfs-api"
)

// payment is a component that holds cyber-gubi. A component is a
// customizable, independent, and reusable UI element. It is created by
// embedding app.Compo into a struct.
type tax struct {
	app.Compo
	sh        *shell.Shell
	loggedIn  bool
	userID    string
	taxUserID string
	taxAmount int
	wallet    Wallet
	taxWallet Wallet
}

func (t *tax) OnMount(ctx app.Context) {
	sh := shell.NewShell("localhost:5001")
	t.sh = sh

	ctx.GetState("loggedIn", &t.loggedIn)
	if !t.loggedIn {
		ctx.Navigate("/auth")
	}

	urlPath := app.Window().URL().Path

	fragments := strings.Split(urlPath, "tax/")

	if len(fragments) > 1 {
		t.taxUserID = fragments[1]
	}

	if t.taxUserID == "" {
		ctx.Navigate("/auth")
	}

	ctx.GetState("userID", &t.userID)

	ctx.GetState("balance", &t.wallet)

	taxWallet, err := t.getBalance(t.taxUserID)
	if err != nil {
		log.Fatal("wallet not found")
	}

	t.taxWallet = taxWallet
}

func (t *tax) getBalance(userID string) (balance Wallet, err error) {
	b, err := t.sh.OrbitDocsQuery(dbWallet, "_id", userID)
	if err != nil {
		return Wallet{}, err
	}

	if len(b) == 0 {
		return Wallet{}, err
	}

	wallets := []Wallet{}

	err = json.Unmarshal(b, &wallets) // Unmarshal the byte slice directly
	if err != nil {
		return Wallet{}, err
	}

	return wallets[0], nil
}

func (t *tax) updateBalance(userID string, balance, income int, date string) error {
	wallet := Wallet{
		ID:           userID,
		Balance:      balance,
		Income:       income,
		LastReceived: date,
	}

	walletJSON, err := json.Marshal(wallet)
	if err != nil {
		return err
	}

	err = t.sh.OrbitDocsPut(dbWallet, walletJSON)
	if err != nil {
		return err
	}

	return nil
}

func (t *tax) storeTransaction(transaction Transaction) error {
	transactionJSON, err := json.Marshal(transaction)
	if err != nil {
		return err
	}

	err = t.sh.OrbitDocsPut(dbTransaction, transactionJSON)
	if err != nil {
		return err
	}

	return nil
}

func (t *tax) withdrawTax(ctx app.Context, e app.Event) {
	e.PreventDefault()

	valid := app.Window().GetElementByID("pay-form").Call("reportValidity").Bool()
	if valid {
		transaction := Transaction{}
		transaction.ID = uuid.NewString()
		transaction.SenderID = t.taxUserID
		transaction.ReceiverID = t.userID
		transaction.TotalCost = t.taxAmount * 100
		transaction.Timestamp = time.Now()
		transaction.Date = strconv.Itoa(time.Now().Year()) + "/" + strconv.Itoa(int(time.Now().Month()))

		if t.taxWallet.Balance-transaction.TotalCost < 0 {
			ctx.Notifications().New(app.Notification{
				Title: "Error",
				Body:  "Not enough funds.",
			})
			return
		}
		// update taxable balance
		err := t.updateBalance(t.taxUserID, t.taxWallet.Balance-transaction.TotalCost, t.taxWallet.Income, t.taxWallet.LastReceived)
		if err != nil {
			log.Fatal(err)
		}
		// get treasury balance
		receiverBalance, err := t.getBalance(t.userID)
		if err != nil {
			log.Fatal(err)
		}
		// update receiver balance
		err = t.updateBalance(transaction.ReceiverID, receiverBalance.Balance+transaction.TotalCost, receiverBalance.Income, receiverBalance.LastReceived)
		if err != nil {
			// rollback sender balance
			err := t.updateBalance(t.taxUserID, t.taxWallet.Balance+transaction.TotalCost, t.taxWallet.Income, t.taxWallet.LastReceived)
			if err != nil {
				log.Fatal(err)
			}
			return
		}
		// store transaction
		err = t.storeTransaction(transaction)
		if err != nil {
			// rollback sender balance
			err = t.updateBalance(t.taxUserID, t.taxWallet.Balance+transaction.TotalCost, t.taxWallet.Income, t.taxWallet.LastReceived)
			if err != nil {
				log.Fatal(err)
			}
			// rollback receiver balance
			err = t.updateBalance(transaction.ReceiverID, receiverBalance.Balance-transaction.TotalCost, receiverBalance.Income, receiverBalance.LastReceived)
			if err != nil {
				log.Fatal(err)
			}
			return
		}

		t.wallet.Balance = t.wallet.Balance + transaction.TotalCost
		ctx.Update()

		ctx.Notifications().New(app.Notification{
			Title: "Success",
			Body:  "Payment successful!",
		})
	}
}

// The Render method is where the component appearance is defined. Here, a
// payment form is displayed.
func (t *tax) Render() app.UI {
	return app.Div().Class("container").Body(
		app.Div().Class("mobile").Body(
			app.Div().Class("header").Body(
				newNav(),
				app.Div().Class("header-summary").Body(
					app.Span().Class("logo").Text("cyber-gubi"),
					app.Div().Class("summary-text").Body(
						app.Span().Text("Treasury"),
					),
					app.Div().Class("summary-balance").Body(
						app.Span().Text(strconv.Itoa(t.wallet.Balance/100)+" GUBI"),
					),
				),
			),
			app.Div().ID("content").Body(
				app.Div().Class("card").Body(
					app.Div().Class("upper-row").Body(
						app.Div().Class("card-item").Body(
							app.Span().Class("span-header").Text("Collect Tax"),
							app.Form().ID("pay-form").Body(
								app.Label().Class("tax-label").Text("Balance"),
								app.Input().Type("number").Value(t.taxWallet.Balance/100).Disabled(true),
								app.Input().Type("number").Placeholder("Amount to collect").Required(true).OnChange(t.ValueTo(&t.taxAmount)),
								app.Div().Class("drawer drawer-pay").Body(
									app.Div().Class("menu-btn").Body(
										app.Button().Class("submit").Type("submit").Text("Withdraw").OnClick(t.withdrawTax),
									),
								),
							),
						),
					),
				),
			),
		),
	)
}
