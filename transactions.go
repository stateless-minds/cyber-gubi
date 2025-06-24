package main

import (
	"encoding/json"
	"log"
	"strconv"
	"strings"

	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"github.com/stateless-minds/kubo/client/rpc"
)

// transactions is a component that holds cyber-gubi. A component is a
// customizable, independent, and reusable UI element. It is created by
// embedding app.Compo into a struct.
type transactions struct {
	app.Compo
	sh           *rpc.HttpApi
	loggedIn     bool
	userID       string
	wallet       Wallet
	transactions []Transaction
	observer     app.Value
	callback     app.Func
	lastIndex    int
	indexStep    int
}

func (t *transactions) OnMount(ctx app.Context) {
	sh, err := rpc.NewLocalApi()
	if err != nil {
		log.Fatal(err)
	}
	t.sh = sh
	t.indexStep = 99

	ctx.GetState("loggedIn", &t.loggedIn)
	if !t.loggedIn {
		ctx.Navigate("/auth")
	}

	urlPath := app.Window().URL().Path

	fragments := strings.Split(urlPath, "transactions/")

	if len(fragments) > 1 {
		t.userID = fragments[1]
	}

	t.callback = app.FuncOf(func(this app.Value, args []app.Value) interface{} {
		entries := args[0]
		for i := 0; i < entries.Length(); i++ {
			entry := entries.Index(i)
			if entry.Get("isIntersecting").Bool() {
				// Element is visible - do something
				t.getTransactions(ctx)
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
	t.observer = observerConstructor.New(t.callback, options)

	if t.userID == "" {
		ctx.GetState("userID", &t.userID)
	}

	ctx.GetState("balance", &t.wallet)

	t.getTransactions(ctx)
}

func (t *transactions) OnUpdate(ctx app.Context) {
	// Wrap your observation logic in a Go function
	callback := func() {
		target := app.Window().GetElementByID("last-item")
		if !target.IsNull() && !target.IsUndefined() {
			t.observer.Call("disconnect")
			t.observer.Call("observe", target)
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

func (t *transactions) OnDismount(ctx app.Context) {
	t.observer.Call("disconnect")
	t.callback.Release()
}

func (t *transactions) getTransactions(ctx app.Context) {
	ctx.Async(func() {
		rangeStart := strconv.Itoa(t.lastIndex)
		rangeEnd := strconv.Itoa(t.lastIndex + t.indexStep)
		// trs, err := t.sh.OrbitDocsQuery(dbTransaction, "all", "")
		trs, err := t.sh.OrbitDocsQuery(dbTransaction, "sender_id,receiver_id", t.userID+",range="+rangeStart+"-"+rangeEnd)
		if err != nil {
			log.Fatal(err)
		}

		transactions := []Transaction{}

		if len(trs) != 0 {
			err = json.Unmarshal(trs, &transactions) // Unmarshal the byte slice directly
			if err != nil {
				log.Fatal(err)
			}
		} else {
			t.OnDismount(ctx)
		}

		ctx.Dispatch(func(ctx app.Context) {
			t.transactions = transactions
			t.lastIndex = t.lastIndex + 1 + t.indexStep
			t.OnUpdate(ctx)
		})
	})
}

func (t *transactions) showTransactionDetails(ctx app.Context, e app.Event) {
	ctx.JSSrc().Call("setAttribute", "style", "height: auto")
}

func (t *transactions) hideTransactionDetails(ctx app.Context, e app.Event) {
	ctx.JSSrc().Call("setAttribute", "style", "height: 55px")
}

func (t *transactions) exportTransactions(ctx app.Context, e app.Event) {
	e.PreventDefault()
	trs, err := json.Marshal(t.transactions)
	if err != nil {
		log.Fatal(err)
	}

	blob := app.Window().Get("Blob").New(
		[]interface{}{string(trs)},
		map[string]interface{}{"type": "application/json"},
	)

	// Create Object URL
	url := app.Window().Get("URL").Call("createObjectURL", blob)

	// Create a temporary <a> element
	a := app.Window().Get("document").Call("createElement", "a")
	a.Set("href", url)
	a.Set("download", "transactions.json")

	// Append to body, click to trigger download, then remove
	app.Window().Get("document").Get("body").Call("appendChild", a)
	a.Call("click")
	a.Get("parentNode").Call("removeChild", a)

	// Revoke the object URL to avoid memory leaks
	app.Window().Get("URL").Call("revokeObjectURL", url)
}

// The Render method is where the component appearance is defined. Here, a
// payment form is displayed.
func (t *transactions) Render() app.UI {
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
						app.Span().Text(strconv.Itoa(t.wallet.Balance/100)+" GUBI"),
					),
				),
			),
			app.Div().ID("content").Body(
				app.Div().Class("card").Body(
					app.Div().Class("upper-row single").Body(
						app.Div().Class("card-item").Body(
							app.Span().Class("span-header-sub").Text("Transactions"),
						),
					),
				),
				app.Div().Class("list transactions").Body(
					app.If(len(t.transactions) == 0, func() app.UI {
						return app.Div().Class("list-item").Body(
							app.Span().Class("empty").Text("No transactions found"),
						).Style("pointer-events", "none")
					}),
					app.Range(t.transactions).Slice(func(i int) app.UI {
						return app.If(i == len(t.transactions)-1 && len(t.transactions)%5 == 0, func() app.UI {
							return app.Div().ID("last-item").Class("list-item").Body(
								app.Div().Class("t-details").Body(
									app.Div().Class("t-title").Body(
										app.If(t.transactions[i].SenderID == t.userID, func() app.UI {
											return app.Span().Text("Purchase ID: " + t.transactions[i].ID)
										}).Else(func() app.UI {
											return app.Span().Text("Sale ID: " + t.transactions[i].ID)
										}),
									),
									app.Div().Class("t-time").Body(
										app.Span().Text(t.transactions[i].Timestamp.Format("2006-01-02 15:04:05")),
									),
									app.Div().Class("t-more-details").Body(
										app.Div().Class("col-1").Body(
											app.Span().Text("Item"),
											app.Range(t.transactions[i].ProductsServices).Slice(func(n int) app.UI {
												return app.Span().Text(t.transactions[i].ProductsServices[n].Name)
											}),
										),
										app.Div().Class("col-2").Body(
											app.Span().Text("Amount"),
											app.Range(t.transactions[i].ProductsServices).Slice(func(n int) app.UI {
												return app.Span().Text(t.transactions[i].ProductsServices[n].Amount)
											}),
										),
										app.Div().Class("col-3").Body(
											app.Span().Text("Price"),
											app.Range(t.transactions[i].ProductsServices).Slice(func(n int) app.UI {
												return app.Span().Text(t.transactions[i].ProductsServices[n].Price / 100)
											}),
										),
									),
								).OnMouseOver(t.showTransactionDetails).OnMouseLeave(t.hideTransactionDetails),
								app.Div().Class("t-price").Body(
									app.If(t.transactions[i].SenderID == t.userID, func() app.UI {
										return app.Span().Text("-" + strconv.Itoa(t.transactions[i].TotalCost/100) + " GUBI")
									}).Else(func() app.UI {
										return app.Span().Text("+" + strconv.Itoa(t.transactions[i].TotalCost/100) + " GUBI")
									}),
								),
							)
						}).Else(func() app.UI {
							return app.Div().Class("list-item").Body(
								app.Div().Class("t-details").Body(
									app.Div().Class("t-title").Body(
										app.If(t.transactions[i].SenderID == t.userID, func() app.UI {
											return app.Span().Text("Purchase ID: " + t.transactions[i].ID)
										}).Else(func() app.UI {
											return app.Span().Text("Sale ID: " + t.transactions[i].ID)
										}),
									),
									app.Div().Class("t-time").Body(
										app.Span().Text(t.transactions[i].Timestamp.Format("2006-01-02 15:04:05")),
									),
									app.Div().Class("t-more-details").Body(
										app.Div().Class("col-1").Body(
											app.Span().Text("Item"),
											app.Range(t.transactions[i].ProductsServices).Slice(func(n int) app.UI {
												return app.Span().Text(t.transactions[i].ProductsServices[n].Name)
											}),
										),
										app.Div().Class("col-2").Body(
											app.Span().Text("Amount"),
											app.Range(t.transactions[i].ProductsServices).Slice(func(n int) app.UI {
												return app.Span().Text(t.transactions[i].ProductsServices[n].Amount)
											}),
										),
										app.Div().Class("col-3").Body(
											app.Span().Text("Price"),
											app.Range(t.transactions[i].ProductsServices).Slice(func(n int) app.UI {
												return app.Span().Text(t.transactions[i].ProductsServices[n].Price / 100)
											}),
										),
									),
								).OnMouseOver(t.showTransactionDetails).OnMouseLeave(t.hideTransactionDetails),
								app.Div().Class("t-price").Body(
									app.If(t.transactions[i].SenderID == t.userID, func() app.UI {
										return app.Span().Text("-" + strconv.Itoa(t.transactions[i].TotalCost/100) + " GUBI")
									}).Else(func() app.UI {
										return app.Span().Text("+" + strconv.Itoa(t.transactions[i].TotalCost/100) + " GUBI")
									}),
								),
							)
						})
					}),
				),
				app.If(len(t.transactions) > 0, func() app.UI {
					return app.Div().Class("menu-btn").Body(
						app.A().Class("submit submit-list").Type("submit").Text("Export").OnClick(t.exportTransactions),
					)
				}),
			),
		),
	)
}
