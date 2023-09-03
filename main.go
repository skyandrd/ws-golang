package main

import (
	"context"
	"encoding/json"
	"flag"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", "localhost:8080", "http service address")

var upgrader = websocket.Upgrader{} // use default options

var clients = map[string]*websocket.Conn{}

func echo(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade error: %v\n", err)
		return
	}
	defer c.Close()

	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Printf("read error: %v\n", err)
			break
		}
		log.Printf("recv: %s\n", message)
		err = c.WriteMessage(mt, message)
		if err != nil {
			log.Printf("write error: %v\n", err)
			break
		}

		clients[string(message)] = c
	}
}

func home(w http.ResponseWriter, r *http.Request) {
	homeTemplate.Execute(w, "ws://"+r.Host+"/echo")
}

type command struct {
	DeviceID string `json:"device_id"`
	ID       string `json:"id"`
	Kind     int    `json:"kind"`
	Message  string `json:"message"`
}

func commandHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()

	if err != nil {
		log.Panicf("commandHandler error %v", err)
	}

	c := command{}
	err = json.Unmarshal(body, &c)
	if err != nil {
		log.Panicf("commandHandler unmarshal error %v", err)
	}

	if client, ok := clients[c.DeviceID]; ok { // send to device id
		err = client.WriteMessage(1, []byte("time now: "+time.Now().Format(time.RFC3339)+"\n"+string(body)))
		if err != nil {
			log.Printf("write error: %v\n", err)
			w.WriteHeader(http.StatusNotFound)
		}
	} else { // send to all devices
		for _, client := range clients {
			err = client.WriteMessage(1, []byte("time now: "+time.Now().Format(time.RFC3339)+"\n"+string(body)))
			if err != nil {
				log.Printf("write error: %v\n", err)
				w.WriteHeader(http.StatusNotFound)
			}
		}
	}

	w.WriteHeader(http.StatusOK)
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/", home).Methods("GET")
	r.HandleFunc("/echo", echo).Methods("GET")
	r.HandleFunc("/command", commandHandler).Methods("POST")

	var wait time.Duration

	flag.DurationVar(&wait, "graceful-timeout", time.Second*15,
		"the duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")
	flag.Parse()

	srv := &http.Server{
		Addr: *addr,
		// Good practice to set timeouts to avoid Slowloris attacks.
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      r, // Pass our instance of gorilla/mux in.
	}

	// Run our server in a goroutine so that it doesn't block.
	go func() {
		log.Println("server starting")
		if err := srv.ListenAndServe(); err != nil {
			log.Printf("ListenAndServe error %v", err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, []os.Signal{syscall.SIGINT, syscall.SIGTERM}...)

	// Block until we receive our signal.
	<-c

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	_ = srv.Shutdown(ctx)
	// Optionally, you could run srv.Shutdown in a goroutine and block on
	// <-ctx.Done() if your application should wait for other services
	// to finalize based on context cancellation.
	log.Println("shutting down")
	os.Exit(0)
}

var homeTemplate = template.Must(template.New("").Parse(`
<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<script> 

function uuidv4() {
	return ([1e7]+-1e3+-4e3+-8e3+-1e11).replace(/[018]/g, c =>
	  (c ^ crypto.getRandomValues(new Uint8Array(1))[0] & 15 >> c / 4).toString(16)
	);
}

window.addEventListener("load", function(evt) {

    var output = document.getElementById("output");
    var input = document.getElementById("input");
    var ws;

    var print = function(message) {
        var d = document.createElement("div");
        d.textContent = message;
        output.appendChild(d);
        output.scroll(0, output.scrollHeight);
    };

    document.getElementById("open").onclick = function(evt) {
        if (ws) {
            return false;
        }
        ws = new WebSocket("{{.}}");
        ws.onopen = function(evt) {
            print("OPEN");

			input.value = uuidv4();
        }
        ws.onclose = function(evt) {
            print("CLOSE");
            ws = null;
        }
        ws.onmessage = function(evt) {
            print("RESPONSE: " + evt.data);
        }
        ws.onerror = function(evt) {
            print("ERROR: " + evt.data);
        }
        return false;
    };

    document.getElementById("send").onclick = function(evt) {
        if (!ws) {
            return false;
        }
        print("SEND: " + input.value);
        ws.send(input.value);
        return false;
    };

    document.getElementById("close").onclick = function(evt) {
        if (!ws) {
            return false;
        }
        ws.close();
        return false;
    };

});
</script>
</head>
<body>
<table>
<tr><td valign="top" width="50%">
<p>Click "Open" to create a connection to the server, 
"Send" to send a message to the server and "Close" to close the connection. 
<p>
<form>
<button id="open">Open</button>
<button id="close">Close</button>
<p><label>device id&nbsp;<input id="input" type="text" value="device id"></label>
<button id="send">Send</button>
</form>
</td><td valign="top" width="50%">
<div id="output" style="max-height: 70vh;overflow-y: scroll;"></div>
</td></tr></table>
</body>
</html>
`))
