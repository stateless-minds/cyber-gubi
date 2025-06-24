package main

import (
	"encoding/json"
	"log"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"github.com/stateless-minds/kubo/client/rpc"
)

const dbTransaction = "transaction"

// payment is a component that holds cyber-gubi. A component is a
// customizable, independent, and reusable UI element. It is created by
// embedding app.Compo into a struct.
type payment struct {
	app.Compo
	sh            *rpc.HttpApi
	loggedIn      bool
	userID        string
	wallet        Wallet
	wallets       []Wallet
	productsIndex []int
	servicesIndex []int
	products      []ProductService
	services      []ProductService
	activeTab     string
}

type Subscription struct {
	ID        string    `mapstructure:"_id" json:"_id" validate:"uuid_rfc4122"`               // Unique identifier for the transaction
	PlanID    string    `mapstructure:"plan_id" json:"plan_id" validate:"uuid_rfc4122"`       // Plan id
	UserID    string    `mapstructure:"user_id" json:"user_id" validate:"uuid_rfc4122"`       // User id
	Price     int       `mapstructure:"price" json:"price" validate:"uuid_rfc4122"`           // Price
	StartDate time.Time `mapstructure:"start_date" json:"start_date" validate:"uuid_rfc4122"` // Start date of subscription
	EndDate   time.Time `mapstructure:"end_date" json:"end_date" validate:"uuid_rfc4122"`     // End date of subscription
}

type Transaction struct {
	ID               string `mapstructure:"_id" json:"_id" validate:"uuid_rfc4122"`                 // Unique identifier for the transaction
	SenderID         string `mapstructure:"sender_id" json:"sender_id" validate:"uuid_rfc4122"`     // Sender user id
	ReceiverID       string `mapstructure:"receiver_id" json:"receiver_id" validate:"uuid_rfc4122"` // Recipient user id
	ProductsServices []ProductService
	TotalCost        int       `mapstructure:"total_cost" json:"total_cost" validate:"uuid_rfc4122"` // Total cost of transaction
	Timestamp        time.Time `mapstructure:"timestamp" json:"timestamp" validate:"uuid_rfc4122"`   // Timestamp of the transaction
	Date             string    `mapstructure:"date" json:"date" validate:"uuid_rfc4122"`             // Date of the transaction in the format YY/MM
	Processed        bool      `mapstructure:"processed" json:"processed" validate:"uuid_rfc4122"`   // Flag if it was already processed by inflation indexer
}

type ProductService struct {
	ID     string `mapstructure:"product_id" json:"product_id" validate:"uuid_rfc4122"` // Unique identifier for the product
	Name   string `mapstructure:"name" json:"name" validate:"uuid_rfc4122"`
	Price  int    `mapstructure:"price" json:"price" validate:"uuid_rfc4122"`
	Amount int    `mapstructure:"amount" json:"amount" validate:"uuid_rfc4122"`
}

func (p *payment) OnMount(ctx app.Context) {
	sh, err := rpc.NewLocalApi()
	if err != nil {
		log.Fatal(err)
	}
	p.sh = sh

	// set default number of product inputs
	p.productsIndex = []int{1}
	p.products = make([]ProductService, 1)
	// set default number of service inputs
	p.servicesIndex = []int{1}
	p.services = make([]ProductService, 1)
	p.activeTab = "product"

	ctx.GetState("loggedIn", &p.loggedIn)
	if !p.loggedIn {
		ctx.Navigate("/auth")
	}

	ctx.GetState("userID", &p.userID)
	ctx.GetState("balance", &p.wallet)

	p.getBalances(ctx)
}

func (p *payment) getBalance(userID string) (balance Wallet, err error) {
	b, err := p.sh.OrbitDocsQuery(dbWallet, "_id", userID)
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

func removeSelfFromUserResults(wallets []Wallet, userID string) []Wallet {
	for i, ub := range wallets {
		if ub.ID == userID {
			return append(wallets[:i], wallets[i+1:]...)
		}
	}
	return wallets
}

func (p *payment) getBalances(ctx app.Context) {
	ctx.Async(func() {
		b, err := p.sh.OrbitDocsQuery(dbWallet, "all", "")
		if err != nil {
			log.Fatal(err)
		}

		if len(b) == 0 {
			log.Fatal(err)
		}

		wallets := []Wallet{}

		err = json.Unmarshal(b, &wallets) // Unmarshal the byte slice directly
		if err != nil {
			log.Fatal(err)
		}

		wallets = removeSelfFromUserResults(wallets, p.userID)

		ctx.Dispatch(func(ctx app.Context) {
			p.wallets = wallets
		})

	})
}

func (p *payment) updateBalance(userID string, balance, income int, date string) error {
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

	err = p.sh.OrbitDocsPut(dbWallet, walletJSON)
	if err != nil {
		return err
	}

	return nil
}

func (p *payment) storeTransaction(transaction Transaction) error {
	transactionJSON, err := json.Marshal(transaction)
	if err != nil {
		return err
	}

	err = p.sh.OrbitDocsPut(dbTransaction, transactionJSON)
	if err != nil {
		return err
	}

	return nil
}

func (p *payment) showProduct(ctx app.Context, e app.Event) {
	e.PreventDefault()
	p.activeTab = "product"
	elems := app.Window().Get("document").Call("querySelectorAll", ".service")
	for i := 0; i < elems.Length(); i++ {
		elems.Index(i).Call("removeAttribute", "required")
	}
	app.Window().GetElementByID("tab-service").Get("classList").Call("remove", "tab-active")
	ctx.JSSrc().Get("classList").Call("add", "tab-active")
	// Hide Service Tab and show Product Tab
	app.Window().GetElementByID("product-tab").Call("setAttribute", "style", "display: block")
	app.Window().GetElementByID("service-tab").Call("setAttribute", "style", "display: none")
}

func (p *payment) showService(ctx app.Context, e app.Event) {
	e.PreventDefault()
	p.activeTab = "service"
	elemsProduct := app.Window().Get("document").Call("querySelectorAll", ".product")
	for i := 0; i < elemsProduct.Length(); i++ {
		elemsProduct.Index(i).Call("removeAttribute", "required")
	}
	app.Window().GetElementByID("tab-product").Get("classList").Call("remove", "tab-active")
	ctx.JSSrc().Get("classList").Call("add", "tab-active")
	// Hide Product Tab and show Service Tab
	app.Window().GetElementByID("product-tab").Call("setAttribute", "style", "display: none; ")
	elems := app.Window().Get("document").Call("querySelectorAll", ".service")
	for i := 0; i < elems.Length(); i++ {
		elems.Index(i).Call("setAttribute", "required", true)
	}
	app.Window().GetElementByID("service-tab").Call("setAttribute", "style", "display: block")
}

func (p *payment) addProduct(ctx app.Context, e app.Event) {
	e.PreventDefault()

	p.products = append(p.products, ProductService{})
	p.productsIndex = append(p.productsIndex, len(p.productsIndex)+1)
}

func (p *payment) removeProduct(ctx app.Context, e app.Event) {
	e.PreventDefault()
	p.productsIndex = p.productsIndex[:len(p.productsIndex)-1]
	p.products = p.products[:len(p.products)-1]
}

func (p *payment) addService(ctx app.Context, e app.Event) {
	e.PreventDefault()

	p.services = append(p.services, ProductService{})
	p.servicesIndex = append(p.servicesIndex, len(p.servicesIndex)+1)
}

func (p *payment) removeService(ctx app.Context, e app.Event) {
	e.PreventDefault()
	p.servicesIndex = p.servicesIndex[:len(p.servicesIndex)-1]
	p.services = p.services[:len(p.services)-1]
}

func (p *payment) doPayment(ctx app.Context, e app.Event) {
	e.PreventDefault()

	valid := app.Window().GetElementByID("pay-form").Call("reportValidity").Bool()
	if valid {
		tabActive := app.Window().Get("document").Call("getElementsByClassName", "tab-active").Index(0).Get("value").String()
		receiverID := app.Window().GetElementByID("receiver-id").Get("value").String()
		transaction := Transaction{}
		transaction.ID = uuid.NewString()
		transaction.SenderID = p.userID
		transaction.ReceiverID = receiverID
		transaction.Timestamp = time.Now()
		transaction.Date = strconv.Itoa(time.Now().Year()) + "/" + strconv.Itoa(int(time.Now().Month()))
		if tabActive == "product" {
			for i, pr := range p.products {
				p.products[i].ID = uuid.NewString()
				p.products[i].Price = pr.Price * 100
			}
			transaction.ProductsServices = p.products
		} else {
			for i, sr := range p.services {
				p.services[i].ID = uuid.NewString()
				p.services[i].Price = sr.Price * 100
			}
			transaction.ProductsServices = p.services
		}

		var totalCost int

		for _, ps := range transaction.ProductsServices {
			totalCost += ps.Price * ps.Amount
		}

		transaction.TotalCost = totalCost

		if p.wallet.Balance-totalCost < 0 {
			ctx.Notifications().New(app.Notification{
				Title: "Error",
				Body:  "Not enough funds.",
			})
			return
		}
		// update sender balance
		err := p.updateBalance(p.userID, p.wallet.Balance-totalCost, p.wallet.Income, p.wallet.LastReceived)
		if err != nil {
			log.Fatal(err)
		}
		// get receiver balance
		receiverBalance, err := p.getBalance(transaction.ReceiverID)
		if err != nil {
			log.Fatal(err)
		}
		// update receiver balance
		err = p.updateBalance(transaction.ReceiverID, receiverBalance.Balance+totalCost, receiverBalance.Income, receiverBalance.LastReceived)
		if err != nil {
			// rollback sender balance
			err := p.updateBalance(p.userID, p.wallet.Balance+totalCost, p.wallet.Income, p.wallet.LastReceived)
			if err != nil {
				log.Fatal(err)
			}
			return
		}
		// store transaction
		err = p.storeTransaction(transaction)
		if err != nil {
			// rollback sender balance
			err = p.updateBalance(p.userID, p.wallet.Balance+totalCost, p.wallet.Income, p.wallet.LastReceived)
			if err != nil {
				log.Fatal(err)
			}
			// rollback receiver balance
			err = p.updateBalance(transaction.ReceiverID, receiverBalance.Balance-totalCost, receiverBalance.Income, receiverBalance.LastReceived)
			if err != nil {
				log.Fatal(err)
			}
			return
		}

		p.wallet.Balance = p.wallet.Balance - totalCost
		ctx.Update()

		ctx.Notifications().New(app.Notification{
			Title: "Success",
			Body:  "Payment successful!",
		})
	}
}

// The Render method is where the component appearance is defined. Here, a
// payment form is displayed.
func (p *payment) Render() app.UI {
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
						app.Span().Text(strconv.Itoa(p.wallet.Balance/100)+" GUBI"),
					),
				),
			),
			app.Div().ID("content").Body(
				app.Div().Class("card").Body(
					app.Div().Class("upper-row").Body(
						app.Div().Class("card-item").Body(
							app.Span().Class("span-header").Text("Make Payment"),
							app.Form().ID("pay-form").Body(
								app.Label().For("receiver-id").Text("Receiver ID:"),
								app.Select().ID("receiver-id").Name("receiver-id").Body(
									app.Range(p.wallets).Slice(func(i int) app.UI {
										return app.Option().Value(p.wallets[i].ID).Text(p.wallets[i].ID)
									}),
								),
								// Tab Navigation
								app.Div().
									Class("tabs").
									Body(
										app.Button().
											ID("tab-product").
											Class("tab-button").
											Class("tab-active").
											Text("Product").
											Value("product").
											OnClick(p.showProduct),
										app.Button().
											ID("tab-service").
											Class("tab-button").
											Text("Service").
											Value("service").
											OnClick(p.showService),
									),

								// Product Tab Content
								app.Div().
									ID("product-tab").
									Class("tab-content").
									Body(
										app.Range(p.productsIndex).Slice(func(i int) app.UI {
											return app.Div().Body(
												app.Input().ID("product-name-"+strconv.Itoa(i)).Class("product").Type("text").Name("product-name").Placeholder("Product name").Required(true).OnChange(p.ValueTo(&p.products[i].Name)),
												app.Input().ID("product-price-"+strconv.Itoa(i)).Class("product").Type("number").Min(1).Name("product-price").Placeholder("Single price").Required(true).OnChange(p.ValueTo(&p.products[i].Price)),
												app.Input().ID("product-amount-"+strconv.Itoa(i)).Class("product").Type("number").Min(1).Name("product-amount").Step(1).Placeholder("Number of products").Required(true).OnChange(p.ValueTo(&p.products[i].Amount)),
											)
										}),
									),
								// Service Tab Content
								app.Div().
									ID("service-tab").
									Class("tab-content").
									Body(
										app.Range(p.servicesIndex).Slice(func(i int) app.UI {
											return app.Div().Body(
												app.Input().ID("service-name").Class("service").Type("text").Name("service-name").Placeholder("Service name").OnChange(p.ValueTo(&p.services[i].Name)),
												app.Input().ID("service-price").Class("service").Type("number").Min(1).Name("service-price").Placeholder("Price per hour").OnChange(p.ValueTo(&p.services[i].Price)),
												app.Input().ID("service-amount").Class("service").Type("number").Min(1).Name("service-amount").Step(1).Placeholder("Number of hours").OnChange(p.ValueTo(&p.services[i].Amount)),
											)
										}),
									).Hidden(true),
								app.If(p.activeTab == "product", func() app.UI {
									return app.Div().Class("menu-btn menu-add-item").Body(
										app.Button().Class("submit").Text("+").OnClick(p.addProduct),
										app.If(len(p.productsIndex) > 1, func() app.UI {
											return app.Button().Class("submit").Text("-").OnClick(p.removeProduct)
										}),
									)
								}).Else(func() app.UI {
									return app.Div().Class("menu-btn menu-add-item").Body(
										app.Button().Class("submit").Text("+").OnClick(p.addService),
										app.If(len(p.servicesIndex) > 1, func() app.UI {
											return app.Button().Class("submit").Text("-").OnClick(p.removeService)
										}),
									)
								}),
								app.Div().Class("drawer drawer-pay").Body(
									app.Div().Class("menu-btn").Body(
										app.Button().Class("submit").Type("submit").Text("Pay").OnClick(p.doPayment),
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
