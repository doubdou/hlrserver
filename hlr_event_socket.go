package hlr

import (
	"fmt"
	"log"

	"github.com/fiorix/go-eventsocket/eventsocket"
)

const subscribedEvents = "events plain CHANNEL_CREATE CHANNEL_DESTROY CUSTOM sofia::register sofia::unregister sofia::expire verto::login verto::client_connect verto::client_disconnect"

func eventChannelCreate(ev *eventsocket.Event) {
	Debug.Println("recv CHANNEL_CREATE")
}

func eventChannelDestroy(ev *eventsocket.Event) {
	Debug.Println("recv CHANNEL_DESTROY")
}

func eventSofiaRegister(ev *eventsocket.Event) {
	Debug.Println("recv CUSTOM sofia::register")
}

func eventSofiaUnregister(ev *eventsocket.Event) {
	Debug.Println("recv CUSTOM sofia::unregister")
}

func eventSofiaExpire(ev *eventsocket.Event) {
	Debug.Println("recv CUSTOM sofia::expire")
}

func eventVertoLogin(ev *eventsocket.Event) {
	Debug.Println("recv CUSTOM verto::login")
	if ev.Get("verto_result_txt") == "Logged in" {
		Debug.Println("verto user logged in")
	}
}

func eventVertoClientConnect(ev *eventsocket.Event) {
	Debug.Println("recv CUSTOM verto::client_connect")
}

func eventVertoClientDisconnect(ev *eventsocket.Event) {
	Debug.Println("recv CUSTOM verto::client_disconnect")
}

func eventCustom(ev *eventsocket.Event) {
	Debug.Println("recv subclass", ev.Get("Event-Subclass"))
	if ev.Get("Event-Subclass") == "sofia::register" {
		eventSofiaRegister(ev)
	} else if ev.Get("Event-Subclass") == "sofia::unregister" {
		eventSofiaUnregister(ev)
	} else if ev.Get("Event-Subclass") == "sofia::expire" {
		eventSofiaExpire(ev)
	} else if ev.Get("Event-Subclass") == "verto::login" {
		eventVertoLogin(ev)
	} else if ev.Get("Event-Subclass") == "verto::client_connect" {
		eventVertoClientConnect(ev)
	} else if ev.Get("Event-Subclass") == "verto::client_disconnect" {
		eventVertoClientDisconnect(ev)
	} else {
		Error.Println("recv CUSTOM unknown event, Event-Subclass", ev.Get("Event-Subclass"))
	}
}

//EventSocketStartup ESL服务启动入口
func EventSocketStartup(addr string, password string) {
	c, err := eventsocket.Dial(addr, password)
	if err != nil {
		log.Fatal(err)
	}
	c.Send(subscribedEvents)
	// c.Send("events json ALL")
	// c.Send(fmt.Sprintf("bgapi originate %s %s", dest, dialplan))
	go func() {
		for {
			ev, err := c.ReadEvent()
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println("\nNew event")
			// ev.PrettyPrint()
			if ev.Get("Event-Name") == "CHANNEL_CREATE" {
				eventChannelCreate(ev)
			} else if ev.Get("Event-Name") == "CHANNEL_DESTROY" {
				eventChannelDestroy(ev)
			} else if ev.Get("Event-Name") == "CUSTOM" {
				eventCustom(ev)
			} else {
				Error.Println("unsubscribed event ", ev.Get("Event-Name"))
			}

		}
		// c.Close()
	}()

}
