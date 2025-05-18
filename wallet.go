package main

import (
	"encoding/json"
	"errors"
	"log"
	"sort"
	"strconv"
	"time"

	"github.com/maxence-charriere/go-app/v10/pkg/app"
	shell "github.com/stateless-minds/go-ipfs-api"
)

const dbIncome = "income"
const dbWallet = "wallet"

// wallet is a component that holds cyber-gubi. A component is a
// customizable, independent, and reusable UI element. It is created by
// embedding app.Compo into a struct.
type wallet struct {
	app.Compo
	sh           *shell.Shell
	loggedIn     bool
	isBusiness   bool
	isGovernment bool
	businessName string
	countryCode  string
	countryName  string
	userID       string
	wallet       Wallet
	income       Income
	transactions []Transaction
}

type Wallet struct {
	ID           string `mapstructure:"_id" json:"_id" validate:"uuid_rfc4122"`                     // Unique identifier for the user
	CountryCode  string `mapstructure:"country_code" json:"country_code" validate:"uuid_rfc4122"`   // Unique identifier for the country
	Balance      int    `mapstructure:"balance" json:"balance" validate:"uuid_rfc4122"`             // Balance of the user in cents
	Income       int    `mapstructure:"income" json:"income" validate:"uuid_rfc4122"`               // Recurring income of the user in cents
	LastReceived string `mapstructure:"last_received" json:"last_received" validate:"uuid_rfc4122"` // Date when basic income was last received
}

type Income struct {
	ID     string `mapstructure:"_id" json:"_id" validate:"uuid_rfc4122"`       // Unique identifier for the income
	Amount int    `mapstructure:"amount" json:"amount" validate:"uuid_rfc4122"` // Amount of the income in cents
	Period string `mapstructure:"period" json:"period" validate:"uuid_rfc4122"` // Period the income is valid for
}

func (w *wallet) OnMount(ctx app.Context) {
	sh := shell.NewShell("localhost:5001")
	w.sh = sh

	ctx.GetState("loggedIn", &w.loggedIn)
	if !w.loggedIn {
		ctx.Navigate("/auth")
	}

	ctx.GetState("userID", &w.userID)

	ctx.GetState("isBusiness", &w.isBusiness)

	ctx.GetState("businessName", &w.businessName)

	ctx.GetState("isGovernment", &w.isGovernment)

	ctx.GetState("countryCode", &w.countryCode)

	ctx.GetState("countryName", &w.countryName)

	w.getBalance(ctx)
}

func (w *wallet) getTransactions(ctx app.Context) {
	ctx.Async(func() {
		t, err := w.sh.OrbitDocsQuery(dbTransaction, "sender_id,receiver_id", w.userID)
		if err != nil {
			log.Fatal(err)
		}

		transactions := []Transaction{}

		if len(t) != 0 {
			err = json.Unmarshal(t, &transactions) // Unmarshal the byte slice directly
			if err != nil {
				log.Fatal(err)
			}
		}

		ctx.Dispatch(func(ctx app.Context) {
			if len(transactions) > 0 {
				sort.Slice(transactions, func(i, j int) bool {
					return transactions[i].Timestamp.After(transactions[j].Timestamp)
				})

				w.transactions = append(w.transactions, transactions...)
			}
		})
	})
}

func (w *wallet) getOwnPlan(ctx app.Context) {
	ctx.Async(func() {
		p, err := w.sh.OrbitDocsQuery(dbPlan, "created_by", w.userID)
		if err != nil {
			log.Fatal(err)
		}

		plans := []Plan{}

		if len(p) != 0 {
			err = json.Unmarshal(p, &plans) // Unmarshal the byte slice directly
			if err != nil {
				log.Fatal(err)
			}
		}

		ctx.Dispatch(func(ctx app.Context) {
			if len(p) > 0 {
				ctx.SetState("plan", plans[0])
			} else {
				ctx.SetState("plan", Plan{})
			}

			w.getTransactions(ctx)
		})
	})
}

func (w *wallet) getBalance(ctx app.Context) {
	ctx.Async(func() {
		b, err := w.sh.OrbitDocsQuery(dbWallet, "_id", w.userID)
		if err != nil {
			log.Fatal(err)
		}

		wallets := []Wallet{}

		if len(b) == 0 {
			ctx.Dispatch(func(ctx app.Context) {
				w.wallet = Wallet{}
				ctx.SetState("balance", w.wallet)
				if !w.isBusiness && !w.isGovernment {
					w.getIncome(ctx)
					return
				} else {
					w.updateBalance(ctx)
				}
			})
			return
		} else {
			err = json.Unmarshal(b, &wallets) // Unmarshal the byte slice directly
			if err != nil {
				log.Fatal(err)
			}
		}

		ctx.Dispatch(func(ctx app.Context) {
			w.wallet = wallets[0]
			ctx.SetState("balance", w.wallet)

			// check if recurring income was received for this month
			if !w.isBusiness && !w.isGovernment && w.wallet.LastReceived != strconv.Itoa(time.Now().Year())+"/"+strconv.Itoa(int(time.Now().Month())) {
				w.getIncome(ctx)
			} else {
				if w.isBusiness {
					w.getOwnPlan(ctx)
				} else {
					w.getTransactions(ctx)
				}
			}
		})
	})
}

func (w *wallet) updateBalance(ctx app.Context) {
	ctx.Async(func() {
		wallet := Wallet{
			ID:           string(w.userID),
			Balance:      w.wallet.Balance,
			Income:       w.income.Amount,
			LastReceived: w.wallet.LastReceived,
			CountryCode:  w.countryCode,
		}

		walletJSON, err := json.Marshal(wallet)
		if err != nil {
			log.Fatal(err)
		}

		err = w.sh.OrbitDocsPut(dbWallet, walletJSON)
		if err != nil {
			log.Fatal(err)
		}

		ctx.Dispatch(func(ctx app.Context) {
			w.getTransactions(ctx)
		})
	})
}

func (w *wallet) getIncome(ctx app.Context) {
	ctx.Async(func() {
		i, err := w.sh.OrbitDocsQuery(dbIncome, "all", "")
		if err != nil {
			log.Fatal(err)
		}

		income := []Income{}

		if len(i) == 0 {
			log.Fatal(errors.New("no income set"))
		}

		err = json.Unmarshal([]byte(i), &income) // Unmarshal the byte slice directly
		if err != nil {
			log.Fatal(err)
		}

		ctx.Dispatch(func(ctx app.Context) {
			for _, inc := range income {
				if inc.Period == strconv.Itoa(time.Now().Year())+"/"+strconv.Itoa(int(time.Now().Month())) {
					w.income = inc
				}
			}

			// check if there is a matching income year and month to current moment
			if w.income.Period == strconv.Itoa(time.Now().Year())+"/"+strconv.Itoa(int(time.Now().Month())) {
				w.wallet.Balance = (w.wallet.Balance + w.income.Amount)
				w.wallet.Income = w.income.Amount
				w.wallet.LastReceived = strconv.Itoa(time.Now().Year()) + "/" + strconv.Itoa(int(time.Now().Month()))
				ctx.SetState("balance", w.wallet)
				w.updateBalance(ctx)
			} else {
				w.getTransactions(ctx)
			}
		})
	})
}

func (w *wallet) goToPayments(ctx app.Context, e app.Event) {
	ctx.Navigate("payment")
}

// The Render method is where the component appearance is defined. Here, a
// wallet is displayed.
func (w *wallet) Render() app.UI {
	return app.Div().Class("container").Body(
		app.Div().Class("mobile").Body(
			app.Div().Class("header").Body(
				newNav(),
				app.Div().Class("header-summary").Body(
					app.Span().Class("logo").Text("cyber-gubi"),
					app.Div().Class("summary-text").Body(
						app.Span().Text("Balance"),
					),
					app.Div().Class("summary-balance").Body(
						app.Span().Text(strconv.Itoa(w.wallet.Balance/100)+" GUBI"),
					),
				),
			),
			app.Div().ID("content").Body(
				app.Div().Class("card card-wallet").Body(
					app.Div().Class("upper-row").Body(
						app.If(w.isBusiness, func() app.UI {
							return app.Div().Class("card-item").Body(
								app.Span().Class("span-header").Text("Business Name"),
								app.Span().Class("span-body").Text(w.businessName),
							)
						}).ElseIf(w.isGovernment, func() app.UI {
							return app.Div().Class("card-item").Body(
								app.Span().Class("span-header").Text("Country Name"),
								app.Span().Class("span-body").Text(w.countryName),
							)
						}).Else(func() app.UI {
							return app.Div().Class("card-item").Body(
								app.Span().Class("span-header").Text("Monthly Recurring"),
								app.Span().Text(strconv.Itoa(w.wallet.Income/100)+" GUBI"),
							)
						}),
					),
					app.Div().Class("lower-row").Body(
						app.Div().Class("card-item").Body(
							app.Span().Class("span-header").Text("Payment ID"),
							app.Span().Class("span-body").Text(w.userID),
						),
					),
				),
				app.Div().Class("menu-btn").Body(
					app.Button().Class("submit").Type("submit").Text("Make a payment").OnClick(w.goToPayments),
				),
			),
		),
	)
}
