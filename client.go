package main

import (
	"encoding/json"
	"log"
	"strconv"

	"github.com/maxence-charriere/go-app/v10/pkg/app"
	shell "github.com/stateless-minds/go-ipfs-api"
)

// client is a component that holds cyber-gubi. A component is a
// customizable, independent, and reusable UI element. It is created by
// embedding app.Compo into a struct.
type client struct {
	app.Compo
	sh            *shell.Shell
	loggedIn      bool
	businessName  string
	wallet        Wallet
	plan          Plan
	subscriptions []Subscription
	totalIncome   int
	observer      app.Value
	callback      app.Func
	lastIndex     int
	indexStep     int
}

func (c *client) OnMount(ctx app.Context) {
	sh := shell.NewShell("localhost:5001")
	c.sh = sh
	c.indexStep = 99

	ctx.GetState("loggedIn", &c.loggedIn)
	if !c.loggedIn {
		ctx.Navigate("/auth")
	}

	c.callback = app.FuncOf(func(this app.Value, args []app.Value) interface{} {
		entries := args[0]
		for i := 0; i < entries.Length(); i++ {
			entry := entries.Index(i)
			if entry.Get("isIntersecting").Bool() {
				// Element is visible - do something
				c.getSubscriptions(ctx)
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
	c.observer = observerConstructor.New(c.callback, options)

	ctx.GetState("businessName", &c.businessName)

	ctx.GetState("balance", &c.wallet)

	ctx.GetState("plan", &c.plan)

	c.getSubscriptions(ctx)
}

func (c *client) OnUpdate(ctx app.Context) {
	// Wrap your observation logic in a Go function
	callback := func() {
		target := app.Window().GetElementByID("last-item")
		if !target.IsNull() && !target.IsUndefined() {
			c.observer.Call("disconnect")
			c.observer.Call("observe", target)
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

func (c *client) OnDismount(ctx app.Context) {
	c.observer.Call("disconnect")
	c.callback.Release()
}

func (c *client) getSubscriptions(ctx app.Context) {
	ctx.Async(func() {
		rangeStart := strconv.Itoa(c.lastIndex)
		rangeEnd := strconv.Itoa(c.lastIndex + c.indexStep)
		subs, err := c.sh.OrbitDocsQuery(dbSubscription, "plan_id", c.plan.ID+",range="+rangeStart+"-"+rangeEnd)
		if err != nil {
			log.Fatal(err)
		}

		subscriptions := []Subscription{}
		var totalIncome int

		if len(subs) != 0 {
			err = json.Unmarshal(subs, &subscriptions) // Unmarshal the byte slice directly
			if err != nil {
				log.Fatal(err)
			}

			for _, sub := range subscriptions {
				totalIncome += sub.Price
			}
		} else {
			c.OnDismount(ctx)
		}

		ctx.Dispatch(func(ctx app.Context) {
			c.subscriptions = append(c.subscriptions, subscriptions...)
			c.totalIncome = totalIncome
			c.lastIndex = c.lastIndex + 1 + c.indexStep
			c.OnUpdate(ctx)
		})
	})
}

// The Render method is where the component appearance is defined. Here, a
// client is displayed.
func (c *client) Render() app.UI {
	return app.Div().Class("container").Body(
		app.Div().Class("mobile").Body(
			app.Div().Class("header").Body(
				newNav(),
				app.Div().Class("header-summary").Body(
					app.Span().Class("logo").Text("cyber-gubi"),
					app.Div().Class("summary-text").Body(
						app.Span().Text("Recurring"),
					),
					app.Div().Class("summary-balance").Body(
						app.Span().Text(strconv.Itoa(c.totalIncome/100)+" GUBI"),
					),
				),
			),
			app.Div().ID("content").Body(
				app.Div().Class("card").Body(
					app.Div().Class("upper-row single").Body(
						app.Div().Class("card-item").Body(
							app.Span().Class("span-header-sub").Text("Clients"),
						),
					),
				),
				app.Div().Class("list").Body(
					app.If(len(c.subscriptions) == 0, func() app.UI {
						return app.Div().Class("list-item").Body(
							app.Span().Class("empty").Text("No subscriptions yet"),
						).Style("pointer-events", "none")
					}),
					app.Range(c.subscriptions).Slice(func(i int) app.UI {
						return app.If(i == len(c.subscriptions)-1 && len(c.subscriptions)%5 == 0, func() app.UI {
							return app.Div().ID("last-item").Class("list-item").Body(
								app.Div().Class("s-details").Body(
									app.Div().Class("c-title").Body(
										app.Span().Text("User ID: "+c.subscriptions[i].UserID),
									),
									app.Div().Class("s-time").Body(
										app.Span().Text(c.subscriptions[i].StartDate.Format("2006-01-02 15:04")),
										app.Span().Text(c.subscriptions[i].EndDate.Format("2006-01-02 15:04")),
									),
								),
								app.Div().Class("s-price").Body(
									app.Span().Text(strconv.Itoa(c.subscriptions[i].Price/100)+" GUBI"),
								),
							)
						}).Else(func() app.UI {
							return app.Div().Class("list-item").Body(
								app.Div().Class("s-details").Body(
									app.Div().Class("c-title").Body(
										app.Span().Text("User ID: "+c.subscriptions[i].UserID),
									),
									app.Div().Class("s-time").Body(
										app.Span().Text(c.subscriptions[i].StartDate.Format("2006-01-02 15:04")),
										app.Span().Text(c.subscriptions[i].EndDate.Format("2006-01-02 15:04")),
									),
								),
								app.Div().Class("s-price").Body(
									app.Span().Text(strconv.Itoa(c.subscriptions[i].Price/100)+" GUBI"),
								),
							)
						})
					}),
				),
			),
		),
	)
}
