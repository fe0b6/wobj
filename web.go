package wobj

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Начитаем слушать порт
func listen(port int) {

	if params.ParseRequest == nil {
		http.HandleFunc("/", parseRequest)
	} else {
		http.HandleFunc("/", params.ParseRequest)
	}

	if params.WsRoute != nil {
		if params.WsPath == "" {
			params.WsPath = "/ws/"
		}
		http.HandleFunc(params.WsPath, wsRequest)
	}

	log.Fatalln("[fatal]", http.ListenAndServe(":"+strconv.Itoa(port), nil))
}

// Разбираем запрос
func parseRequest(w http.ResponseWriter, r *http.Request) {

	// Если сервер завершает работу
	if exited {
		w.WriteHeader(503)
		w.Write([]byte(http.StatusText(503)))
		return
	}

	// Отмечаем что начался новый запрос
	wg.Add(1)
	// По завершению запроса отмечаем что он закончился
	defer wg.Done()

	o := &Obj{R: r, W: w, TimeStart: time.Now(), Cache: make(map[string]interface{})}
	params.Route(o)
}

func wsRequest(w http.ResponseWriter, r *http.Request) {
	// Если сервер завершает работу
	if exited {
		w.WriteHeader(503)
		w.Write([]byte(http.StatusText(503)))
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("[error]", err)
		return
	}
	defer conn.Close()

	// Отмечаем что начался новый запрос
	wg.Add(1)
	// По завершению запроса отмечаем что он закончился
	defer wg.Done()

	// Если выходим
	go func(conn *websocket.Conn) {
		_ = <-wsChan
		f := conn.CloseHandler()
		err = f(521, http.StatusText(521))
		if err != nil {
			log.Println("[error]", err)
			return
		}
		conn.Close()
	}(conn)

	params.WsRoute(r, conn)
}

// SendAnswer - функция отправки ответа
func (wo *Obj) SendAnswer() {
	// Если ничего не надо делать
	if wo.Ans.Exited {
		return
	}

	// Если нужно вернуть код
	if wo.Ans.Code != 0 {
		wo.sendCode()
		return
	}

	// Добавляем куку если надо
	if wo.Ans.Cookie != "" {
		cookie := http.Cookie{
			Name:     params.Cookie.Name,
			Domain:   params.Cookie.Domain,
			Path:     params.Cookie.Path,
			Value:    wo.Ans.Cookie,
			MaxAge:   params.Cookie.Time,
			HttpOnly: true,
			Secure:   params.Cookie.Secure,
		}
		http.SetCookie(wo.W, &cookie)
	}

	// Если переадресация
	if wo.Ans.Redirect != "" {
		wo.W.Header().Add("Expires", "Thu, 01 Jan 1970 00:00:01 GMT")
		http.Redirect(wo.W, wo.R, wo.Ans.Redirect, 301)
		return
	}

	// Формируем ответ
	str, err := wo.Tmpl()
	if err != nil {
		log.Println("[error]", err)
		wo.Ans.Code = 500
		wo.sendCode()
		return
	}

	// Добавляем csp
	if params.Csp != "" {
		wo.W.Header().Add("Content-Security-Policy", params.Csp)
	}

	wo.W.Write([]byte(str))
}

// Отправляем ответ
func (wo *Obj) sendCode() {
	wo.W.WriteHeader(wo.Ans.Code)
	// Если не 200 то добавляем статус ответа
	if wo.Ans.Code != 200 {
		wo.W.Write([]byte(http.StatusText(wo.Ans.Code)))
	}
}
