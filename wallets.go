package main

import (
	"encoding/json"
	"log"
	"strconv"

	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"github.com/stateless-minds/kubo/client/rpc"
)

// wallets is a component that holds cyber-gubi. A component is a
// customizable, independent, and reusable UI element. It is created by
// embedding app.Compo into a struct.
type wallets struct {
	app.Compo
	sh          *rpc.HttpApi
	loggedIn    bool
	userID      string
	countryCode string
	wallet      Wallet
	wallets     []Wallet
	observer    app.Value
	callback    app.Func
	lastIndex   int
	indexStep   int
}

func (w *wallets) OnMount(ctx app.Context) {
	sh, err := rpc.NewLocalApi()
	if err != nil {
		log.Fatal(err)
	}
	w.sh = sh
	w.indexStep = 99

	ctx.GetState("loggedIn", &w.loggedIn)
	if !w.loggedIn {
		ctx.Navigate("/auth")
	}

	w.callback = app.FuncOf(func(this app.Value, args []app.Value) interface{} {
		entries := args[0]
		for i := 0; i < entries.Length(); i++ {
			entry := entries.Index(i)
			if entry.Get("isIntersecting").Bool() {
				// Element is visible - do something
				w.getWallets(ctx)
			}
		}
		return nil
	})

	// Select the root element by class name
	rootElement := app.Window().Get("document").Call("querySelector", ".list")

	options := map[string]interface{}{
		"root":       rootElement,
		"rootMargin": "0px",
		"threshold":  1,
	}

	observerConstructor := app.Window().Get("IntersectionObserver")
	w.observer = observerConstructor.New(w.callback, options)

	ctx.GetState("userID", &w.userID)
	ctx.GetState("balance", &w.wallet)

	ctx.GetState("countryCode", &w.countryCode)

	w.getWallets(ctx)
}

func (w *wallets) OnUpdate(ctx app.Context) {
	// Wrap your observation logic in a Go function
	callback := func() {
		target := app.Window().GetElementByID("last-item")
		if !target.IsNull() && !target.IsUndefined() {
			w.observer.Call("disconnect")
			w.observer.Call("observe", target)
		}
	}

	var goFunc app.Func

	// Wrap callback as JS function
	goFunc = app.FuncOf(func(this app.Value, args []app.Value) interface{} {
		callback()
		goFunc.Release() // release after call to avoid leaks
		return nil
	})

	// Call JS setTimeout with delay 10ms
	app.Window().Call("goAppSetTimeout", goFunc, 100)
}

func (w *wallets) OnDismount(ctx app.Context) {
	w.observer.Call("disconnect")
	w.callback.Release()
}

func (w *wallets) getWallets(ctx app.Context) {
	ctx.Async(func() {
		rangeStart := strconv.Itoa(w.lastIndex)
		rangeEnd := strconv.Itoa(w.lastIndex + w.indexStep)
		// wls, err := w.sh.OrbitDocsQuery(dbWallet, "country_code", w.countryCode+",range="+rangeStart+"-"+rangeEnd)
		wls, err := w.sh.OrbitDocsQuery(dbWallet, "country_code", "BG"+",range="+rangeStart+"-"+rangeEnd)
		if err != nil {
			log.Fatal(err)
		}

		wallets := []Wallet{}

		if len(wls) != 0 {
			err = json.Unmarshal(wls, &wallets) // Unmarshal the byte slice directly
			if err != nil {
				log.Fatal(err)
			}
		} else {
			w.OnDismount(ctx)
		}

		excludingOwnWallet := []Wallet{}

		for _, wallet := range wallets {
			if wallet.ID != string(w.userID) {
				excludingOwnWallet = append(excludingOwnWallet, wallet)
			}
		}

		ctx.Dispatch(func(ctx app.Context) {
			w.wallets = append(w.wallets, excludingOwnWallet...)
			w.lastIndex = w.lastIndex + 1 + w.indexStep
			w.OnUpdate(ctx)
		})
	})
}

func (w *wallets) getBalance(userID string) (balance Wallet, err error) {
	b, err := w.sh.OrbitDocsQuery(dbWallet, "_id", userID)
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

func (w *wallets) updateBalance(userID string, balance, income int, date string) error {
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

	err = w.sh.OrbitDocsPut(dbWallet, walletJSON)
	if err != nil {
		return err
	}

	return nil
}

func (w *wallets) storeTransaction(transaction Transaction) error {
	transactionJSON, err := json.Marshal(transaction)
	if err != nil {
		return err
	}

	err = w.sh.OrbitDocsPut(dbTransaction, transactionJSON)
	if err != nil {
		return err
	}

	return nil
}

// The Render method is where the component appearance is defined. Here, a
// payment form is displayed.
func (w *wallets) Render() app.UI {
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
						app.Span().Text(strconv.Itoa(w.wallet.Balance/100)+" GUBI"),
					),
				),
			),
			app.Div().ID("content").Body(
				app.Div().Class("card").Body(
					app.Div().Class("upper-row single").Body(
						app.Div().Class("card-item").Body(
							app.Span().Class("span-header-sub").Text("Wallets"),
						),
					),
				),
				app.Div().Class("list").Body(
					app.If(len(w.wallets) == 0, func() app.UI {
						return app.Div().Class("list-item").Body(
							app.Span().Class("empty").Text("No wallets found"),
						).Style("pointer-events", "none")
					}),
					app.Range(w.wallets).Slice(func(i int) app.UI {
						return app.If(i == len(w.wallets)-1 && len(w.wallets)%5 == 0, func() app.UI {
							return app.Div().ID("last-item").Class("list-item").Body(
								app.Div().Class("w-details").Body(
									app.Div().Class("w-title").Body(
										app.Span().Text(w.wallets[i].ID),
									),
									app.Div().Class("menu-btn menu-list").Body(
										app.A().Href("transactions").Class("submit submit-list").Type("submit").Text("Transactions"),
									),
									app.Div().Class("menu-btn menu-list").Body(
										app.A().Href("tax/"+w.wallets[i].ID).Class("submit submit-list").Type("submit").Text("Collect Tax"),
									),
								),
							)
						}).Else(func() app.UI {
							return app.Div().Class("list-item").Body(
								app.Div().Class("w-details").Body(
									app.Div().Class("w-title").Body(
										app.Span().Text(w.wallets[i].ID),
									),
									app.Div().Class("menu-btn menu-list").Body(
										app.A().Href("transactions/"+w.wallets[i].ID).Class("submit submit-list").Type("submit").Text("Transactions"),
									),
									app.Div().Class("menu-btn menu-list").Body(
										app.A().Href("tax/"+w.wallets[i].ID).Class("submit submit-list").Type("submit").Text("Collect Tax"),
									),
								),
							)
						})
					}),
				),
			),
		),
	)
}
