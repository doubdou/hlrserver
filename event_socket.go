package hlr

import (
	"strings"
	"time"

	"github.com/fiorix/go-eventsocket/eventsocket"
)

const subscribedEvents = "events plain CHANNEL_CREATE CHANNEL_DESTROY CUSTOM sofia::register sofia::unregister sofia::expire verto::login verto::client_connect verto::client_disconnect"

func eventChannelCreate(ev *eventsocket.Event) {
	presenceID := ev.Get("Channel-Presence-Id")
	if presenceID == "" {
		return
	}
	Debug.Println("recv [CHANNEL_CREATE] Channel-Presence-Id:", presenceID)
	pos := strings.Index(presenceID, "@")
	if pos < 0 {
		Error.Println("[CHANNEL_CREATE] Channel-Presence-Id is invalid")
		return
	}
	username := presenceID[:pos]
	realm := presenceID[pos+1:]
	userTalkingSet(username, realm, true)
}

func eventChannelDestroy(ev *eventsocket.Event) {
	presenceID := ev.Get("Channel-Presence-Id")
	if presenceID == "" {
		return
	}
	Debug.Println("recv [CHANNEL_DESTROY] Channel-Presence-Id:", presenceID)
	pos := strings.Index(presenceID, "@")
	if pos < 0 {
		Error.Println("[CHANNEL_DESTROY] Channel-Presence-Id is invalid")
		return
	}
	username := presenceID[:pos]
	realm := presenceID[pos+1:]
	userTalkingSet(username, realm, false)
}

func eventSofiaRegister(ev *eventsocket.Event) {

	username := ev.Get("Username")
	realm := ev.Get("Realm")
	host := ev.Get("Network-Ip")
	port := ev.Get("Network-Port")
	Info.Printf("recv [CUSTOM sofia::register] %s@%s from %s:%s", username, realm, host, port)
	userStatusSet(username, realm, StatusAvailable)
	return
}

func eventSofiaUnregister(ev *eventsocket.Event) {
	username := ev.Get("Username")
	realm := ev.Get("Realm")
	host := ev.Get("Network-Ip")
	port := ev.Get("Network-Port")

	Debug.Printf("recv [CUSTOM sofia::unregister] %s@%s from %s:%s", username, realm, host, port)
	userStatusSet(username, realm, StatusLoggedOut)
	return
}

func eventSofiaExpire(ev *eventsocket.Event) {
	Debug.Println("recv [CUSTOM sofia::expire]")
}

func eventVertoLogin(ev *eventsocket.Event) {
	res := ev.Get("Verto_Result_Txt")
	userInfo := ev.Get("Verto_Login")
	pos := strings.Index(userInfo, "@")
	if pos < 0 {
		Error.Println("[CUSTOM verto::login] Channel-Presence-Id is invalid")
		return
	}
	username := userInfo[:pos]
	realm := userInfo[pos+1:]
	cltAddr := ev.Get("Verto_Client_Address")

	if res != "Logged in" {
		Error.Printf("recv [CUSTOM verto::login] %s %s from %s", userInfo, res, cltAddr)
		return
	}
	Debug.Printf("recv [CUSTOM verto::login] %s %s from %s", userInfo, res, cltAddr)
	userStatusSet(username, realm, StatusAvailable)
}

func eventVertoClientConnect(ev *eventsocket.Event) {
	cltAddr := ev.Get("Verto_Client_Address")
	Debug.Printf("recv [CUSTOM verto::client_connect] client connect from %s", cltAddr)
}

func eventVertoClientDisconnect(ev *eventsocket.Event) {
	userInfo := ev.Get("Verto_Login")
	cltAddr := ev.Get("Verto_Client_Address")
	pos := strings.Index(userInfo, "@")
	if pos < 0 {
		Error.Println("[CUSTOM verto::client_disconnect] [Verto_Login] value invalid", userInfo)
		return
	}
	username := userInfo[:pos]
	realm := userInfo[pos+1:]

	Debug.Printf("recv [CUSTOM verto::client_disconnect] %s from %s", userInfo, cltAddr)
	userStatusSet(username, realm, StatusLoggedOut)
}

func eventCustom(ev *eventsocket.Event) {
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

func eventSocketConnect(addr string, password string) (*eventsocket.Connection, error) {
	var c *eventsocket.Connection
	var err error
	//连接FreeSWITCH
	for {
		c, err = eventsocket.Dial(addr, password)
		if err != nil {
			Error.Println(err)
			time.Sleep(5 * time.Second)
		} else {
			c.Send(subscribedEvents)
			break
		}
	}
	return c, err
}

//EventSocketStartup ESL服务启动入口
func EventSocketStartup(addr string, password string) {
	var c *eventsocket.Connection
	// c.Send("events json ALL")
	// c.Send(fmt.Sprintf("bgapi originate %s %s", dest, dialplan))
	go func() {
		//连接FreeSWITCH
		c, _ = eventSocketConnect(addr, password)
		Info.Println("Connected to FreeSWITCH successfully:", addr)
		//进入event的循环
		for {
			ev, err := c.ReadEvent()
			if err != nil {
				Error.Println("event socket disconnected", err)
				c, _ = eventSocketConnect(addr, password)
				Info.Println("reconnected to FreeSWITCH successfully")
				continue
			}
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
