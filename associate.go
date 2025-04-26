package main

import (
	"encoding/json"
	"log"
	"slices"

	"github.com/maxence-charriere/go-app/v10/pkg/app"
	shell "github.com/stateless-minds/go-ipfs-api"
)

// supplier is a component that holds cyber-gubi. A component is a
// customizable, independent, and reusable UI element. It is created by
// embedding app.Compo into a struct.
type associate struct {
	app.Compo
	sh               *shell.Shell
	loggedIn         bool
	userID           string
	associateName    string
	newAssociateName string
	associates       []string
	currentUser      User
	descriptor       string
}

func (a *associate) OnMount(ctx app.Context) {
	sh := shell.NewShell("localhost:5001")
	a.sh = sh

	ctx.GetState("loggedIn", &a.loggedIn)
	if !a.loggedIn {
		ctx.Navigate("/auth")
	}

	ctx.GetState("userID", &a.userID)

	ctx.GetState("associateName", &a.associateName)

	ctx.GetState("currentUser", &a.currentUser)

	a.getAssociates()
}

func (a *associate) getAssociates() {

	associateNames := []string{}

	var descriptor map[string]map[string][]string

	err := json.Unmarshal(a.currentUser.Descriptor, &descriptor)
	if err != nil {
		log.Fatal(err)
	}

	for name := range descriptor {
		if name != a.associateName {
			associateNames = append(associateNames, name)
		}
	}
	a.associates = associateNames
}

func (a *associate) addAssociate(ctx app.Context, e app.Event) {
	e.PreventDefault()
	valid := app.Window().GetElementByID("associate-form").Call("reportValidity").Bool()
	if valid {
		ctx.SetState("newAssociateName", a.newAssociateName).Persist()

		ctx.Notifications().New(app.Notification{
			Title: "Action required",
			Body:  "Associate " + a.newAssociateName + " needs to sit in front of the web camera. Click this notification when ready.",
			Path:  "auth",
		})
	}
}

func (a *associate) removeAssociate(ctx app.Context, e app.Event) {
	e.PreventDefault()
	name := ctx.JSSrc().Get("value").String()

	var descriptor map[string]map[string][]string

	err := json.Unmarshal(a.currentUser.Descriptor, &descriptor)
	if err != nil {
		log.Fatal(err)
	}

	for associate := range descriptor {
		if name == associate {
			a.updateUser(ctx, name)
		}
	}

}

func (a *associate) updateUser(ctx app.Context, name string) {
	ctx.Async(func() {
		user := a.currentUser

		var descriptor map[string]map[string][]string

		err := json.Unmarshal(a.currentUser.Descriptor, &descriptor)
		if err != nil {
			log.Fatal(err)
		}

		delete(descriptor, name)
		userJSON, err := json.Marshal(user)
		if err != nil {
			log.Fatal(err)
		}

		err = a.sh.OrbitDocsPut(dbUser, userJSON)
		if err != nil {
			log.Fatal(err)
		}

		ctx.Dispatch(func(ctx app.Context) {
			delete(descriptor, name)
			for i, associate := range a.associates {
				if associate == name {
					a.associates = slices.Delete(a.associates, i, i+1)
				}
			}
			ctx.Notifications().New(app.Notification{
				Title: "Success",
				Body:  "Associate " + a.newAssociateName + " has been deleted.",
			})
		})
	})
}

// The Render method is where the component appearance is defined. Here, a
// payment form is displayed.
func (a *associate) Render() app.UI {
	return app.Div().Class("container").Body(
		app.Div().Class("mobile").Body(
			app.Div().Class("header").Body(
				newNav(),
				app.Div().Class("header-summary").Body(
					app.Span().Class("logo").Text("cyber-gubi"),
					app.Div().Class("summary-text").Body(
						app.Span().Text("Associates"),
					),
				),
			),
			app.Div().ID("content").Body(
				app.Div().Class("card").Body(
					app.Div().Class("upper-row").Body(
						app.Div().Class("card-item").Body(
							app.Span().Class("span-header").Text("Add Associate"),
							app.Form().ID("associate-form").Body(
								app.Div().ID("associate").Body(
									app.Input().ID("associate-name").Type("text").Name("associate-name").Placeholder("Associate name").Required(true).OnChange(a.ValueTo(&a.newAssociateName)),
								),
								app.Div().Class("drawer drawer-pay").Body(
									app.Div().Class("menu-btn").Body(
										app.Button().Class("submit").Type("submit").Text("Submit").OnClick(a.addAssociate),
									),
								),
							),
						),
					),
				),
				app.Div().Class("associates").Body(
					app.Span().Class("a-desc").Text("Manage Associates"),
					app.If(len(a.associates) == 0, func() app.UI {
						return app.Div().Class("subscription").Body(
							app.Span().Class("empty").Text("No associates yet"),
						).Style("pointer-events", "none")
					}),
					app.Range(a.associates).Slice(func(i int) app.UI {
						return app.Div().Class("associate").Body(
							app.Div().Class("a-details").Body(
								app.Div().Class("a-title").Body(
									app.Span().Text(a.associates[i]),
								),
							),
							app.Div().Class("a-price").Body(
								app.Div().Class("menu-btn menu-assoc").Body(
									app.Button().Class("submit submit-sub").Type("submit").Text("Remove").Value(a.associates[i]).OnClick(a.removeAssociate),
								),
							),
						)
					}),
				),
			),
		),
	)
}
