package main

import (
	"encoding/json"
	"log"
	"strconv"

	"github.com/google/uuid"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"github.com/stateless-minds/kubo/client/rpc"
)

const dbPlan = "plan"

// plan is a component that holds cyber-gubi. A component is a
// customizable, independent, and reusable UI element. It is created by
// embedding app.Compo into a struct.
type plan struct {
	app.Compo
	sh           *rpc.HttpApi
	loggedIn     bool
	userID       string
	businessName string
	price        int
	plan         Plan
}

type Plan struct {
	ID        string `mapstructure:"_id" json:"_id" validate:"uuid_rfc4122"`               // Unique identifier for the transaction
	Name      string `mapstructure:"name" json:"name" validate:"uuid_rfc4122"`             // Business name
	Price     int    `mapstructure:"price" json:"price" validate:"uuid_rfc4122"`           // Monthly recurring price
	CreatedBy string `mapstructure:"created_by" json:"created_by" validate:"uuid_rfc4122"` // User ID of business who created it
}

func (p *plan) OnMount(ctx app.Context) {
	sh, err := rpc.NewLocalApi()
	if err != nil {
		log.Fatal(err)
	}
	p.sh = sh

	ctx.GetState("loggedIn", &p.loggedIn)
	if !p.loggedIn {
		ctx.Navigate("/auth")
	}

	ctx.GetState("userID", &p.userID)

	ctx.ObserveState("plan", &p.plan)

	ctx.ObserveState("businessName", &p.businessName)
}

func (p *plan) createPlan(ctx app.Context, e app.Event) {
	e.PreventDefault()
	valid := app.Window().GetElementByID("plan-form").Call("reportValidity").Bool()
	if valid {
		p.storePLan(ctx)
	}
}

func (p *plan) storePLan(ctx app.Context) {
	ctx.Async(func() {
		var plan Plan
		if (p.plan == Plan{}) {
			plan = Plan{
				ID:        uuid.NewString(),
				Name:      p.businessName,
				Price:     p.price * 100,
				CreatedBy: p.userID,
			}
		} else {
			plan = Plan{
				ID:        p.plan.ID,
				Name:      p.businessName,
				Price:     p.price * 100,
				CreatedBy: p.plan.CreatedBy,
			}
		}

		planJSON, err := json.Marshal(plan)
		if err != nil {
			log.Fatal(err)
		}

		err = p.sh.OrbitDocsPut(dbPlan, planJSON)
		if err != nil {
			log.Fatal(err)
		}

		ctx.Dispatch(func(ctx app.Context) {
			if (p.plan == Plan{}) {
				ctx.Notifications().New(app.Notification{
					Title: "Success",
					Body:  "Plan created successfully!",
				})
			} else {
				ctx.Notifications().New(app.Notification{
					Title: "Success",
					Body:  "Plan updated successfully!",
				})
			}
			ctx.Navigate("/wallet")
		})
	})
}

// The Render method is where the component appearance is defined. Here, a
// create plan form is displayed.
func (p *plan) Render() app.UI {
	return app.Div().Class("container").Body(
		app.Div().Class("mobile").Body(
			app.Div().Class("header").Body(
				newNav(),
				app.Div().Class("header-summary").Body(
					app.Span().Class("logo").Text("cyber-gubi"),
					app.Div().Class("summary-text").Body(
						app.Span().Text("Plan"),
					),
				),
			),
			app.Div().ID("content").Body(
				app.Div().Class("card").Body(
					app.Div().Class("upper-row").Body(
						app.Div().Class("card-item").Body(
							app.If(p.plan == Plan{}, func() app.UI {
								return app.Span().Class("span-header").Text("Create Plan")
							}).Else(func() app.UI {
								return app.Span().Class("span-header").Text("Update Plan")
							}),
							app.Form().ID("plan-form").Body(
								app.Div().ID("plan").Body(
									app.If(p.plan == Plan{}, func() app.UI {
										return app.Div().Body(
											app.Input().ID("plan-price").Class("product").Type("number").Min(1).Name("plan-price").Placeholder("Monthly amount").Required(true).OnChange(p.ValueTo(&p.price)),
										)
									}).Else(func() app.UI {
										return app.Div().Body(
											app.Input().ID("plan-price").Class("product").Type("number").Min(1).Name("plan-price").Placeholder(strconv.Itoa(p.plan.Price / 100)).Required(true).OnChange(p.ValueTo(&p.price)),
										)
									}),
								),
								app.Div().Class("drawer drawer-pay").Body(
									app.Div().Class("menu-btn").Body(
										app.Button().Class("submit").Type("submit").Text("Submit").OnClick(p.createPlan),
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
