package main

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// Named types
type (
	buns         int
	meat         int
	salad        int
	burger       int
	veggieBurger int
	Seat         int
)

type Customer struct {
	number int
}

// Kitchen tools
type kitchenTool struct {
	sync.Mutex
	name string
}

func newKitchenTool(name string) *kitchenTool {
	return &kitchenTool{name: name}
}

func toastBuns(toaster *kitchenTool) buns {
	toaster.Lock()
	defer toaster.Unlock()
	time.Sleep(time.Microsecond * time.Duration(500))
	return buns(0)
}

func grillMeat(grill *kitchenTool) meat {
	grill.Lock()
	defer grill.Unlock()
	time.Sleep(time.Second * time.Duration(1))
	return meat(0)
}

func chopSalad(chopper *kitchenTool) salad {
	chopper.Lock()
	defer chopper.Unlock()
	time.Sleep(time.Microsecond * time.Duration(200))
	return salad(0)
}

// Order
type order struct {
	buns         buns
	salad        salad
	veggieBurger chan veggieBurger
	meat         chan meat
}

// Serving pipeline
type servingPipeline struct {
	toaster *kitchenTool
	grill   *kitchenTool
	chopper *kitchenTool

	veggieBurgerOrders   chan order
	ordersContainingBuns chan order
	veggieBurgerCookDone chan int

	meatOrders   chan order
	meatCookDone chan int
}

func newServingPipeline(buffer int) *servingPipeline {
	pipeline := &servingPipeline{
		toaster: newKitchenTool("toaster"),
		grill:   newKitchenTool("grill"),
		chopper: newKitchenTool("chopper"),

		veggieBurgerOrders:   make(chan order, buffer),
		ordersContainingBuns: make(chan order, 5), // custom value can be passed
		veggieBurgerCookDone: make(chan int),

		meatOrders:   make(chan order, buffer),
		meatCookDone: make(chan int),
	}
	go pipeline.toastingCook()
	go pipeline.saladCook()
	go pipeline.grillCook()

	return pipeline
}

func (pipeline *servingPipeline) cookBurger(outputChannel chan<- burger) {
	order := order{veggieBurger: make(chan veggieBurger, 1), meat: make(chan meat, 1)}
	pipeline.veggieBurgerOrders <- order
	pipeline.meatOrders <- order
	veggieBurger := <-order.veggieBurger
	burgerMeat := <-order.meat
	outputChannel <- burger(int(veggieBurger) + int(burgerMeat))
}

func (pipeline *servingPipeline) toastingCook() {
	defer close(pipeline.ordersContainingBuns)
	for order := range pipeline.veggieBurgerOrders {
		order.buns = toastBuns(pipeline.toaster)
		pipeline.ordersContainingBuns <- order
	}
}

func (pipeline *servingPipeline) saladCook() {
	defer close(pipeline.veggieBurgerCookDone)
	for order := range pipeline.ordersContainingBuns {
		order.salad = chopSalad(pipeline.chopper)
		order.veggieBurger <- veggieBurger(int(order.salad) + int(order.buns))
	}
}

func (pipeline *servingPipeline) grillCook() {
	close(pipeline.meatCookDone)
	for order := range pipeline.meatOrders {
		order.meat <- grillMeat(pipeline.grill)
	}
}

// Bar
type Bar struct {
	opened          bool
	capacity        int
	seats           chan Seat
	servingPipeline *servingPipeline
}

func newBar(capacity int) *Bar {
	bar := &Bar{
		capacity:        capacity,
		seats:           make(chan Seat, capacity),
		servingPipeline: newServingPipeline(capacity),
	}

	for i := 0; i < cap(bar.seats); i++ {
		bar.seats <- Seat(i)
	}

	return bar
}

func (bar Bar) LogMessage(customer Customer, msg string) {
	log.Println("[CUSTOMER NO.", customer.number, "]: ", msg)
}

func (bar Bar) OrderBurger() <-chan burger {
	receiverChannel := make(chan burger, 1)
	go func() {
		bar.servingPipeline.cookBurger(receiverChannel)
	}()
	return receiverChannel
}

func (bar Bar) ServeCustomer(customer Customer, seat Seat) {
	bar.LogMessage(customer, fmt.Sprintf("Occupied seat %d", seat))
	bar.LogMessage(customer, "Send order")
	burgerChannel := bar.OrderBurger()
	bar.LogMessage(customer, "Order registered - waiting order fulfillment")
	<-burgerChannel
	bar.LogMessage(customer, "Order received - eating")
	time.Sleep(time.Millisecond * time.Duration(500))
	bar.LogMessage(customer, fmt.Sprintf("Leaving the seat %d", seat))
	bar.seats <- seat
}

func main() {
	bar := newBar(4)

	for i := 0; ; i++ {
		customer := Customer{number: i}
		log.Println("Customer NO.", i, "waiting to enter the bar.")
		seat, ok := <-bar.seats
		if ok {
			log.Println("Customer NO.", i, "enters the bar.")
			go bar.ServeCustomer(customer, seat)
		}
	}
}
