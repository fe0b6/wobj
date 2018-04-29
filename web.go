package wobj

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

const (
	wsWriteTimeout = 10 * time.Second
	wsPongWait     = 60 * time.Second
	wsPingPeriod   = 50 * time.Second // Должно быть меньше wsPongWait
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

	if perfomanceFh != nil {
		go func(o *Obj) {
			perfomanceLock.Lock()
			defer perfomanceLock.Unlock()

			dur := (time.Now().UnixNano() - o.TimeStart.UnixNano()) / 1000000
			b, err := json.Marshal(perfomanceData{Path: r.URL.Path, Duration: dur})
			if err != nil {
				log.Println("[error]", err)
				return
			}
			perfomanceFh.Write(b)
			perfomanceFh.WriteString("\n")
		}(o)
	}
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

	// Отмечаем что начался новый запрос
	wg.Add(1)

	ws := WsConn{
		Conn:   conn,
		Reader: make(chan []byte, 10),
		Writer: make(chan []byte, 10),
		Close:  make(chan bool),
	}

	// Добавляем время ожидания закрытия канала
	ws.Conn.SetReadDeadline(time.Now().Add(wsPongWait))
	ws.Conn.SetPongHandler(func(string) error { conn.SetReadDeadline(time.Now().Add(wsPongWait)); return nil })

	// Если выходим
	go func(ws *WsConn) {
		defer wg.Done()
		defer ws.Conn.Close()

		// Ждем сигнала на выход
		select {
		case _ = <-wsChan:
		case _ = <-ws.Close:
		}

		select {
		case <-ws.Reader:
		default:
			close(ws.Reader)
		}
		select {
		case <-ws.Close:
		default:
			close(ws.Close)
		}

		f := ws.Conn.CloseHandler()
		err = f(521, http.StatusText(521))
		if err != nil {
			log.Println("[error]", err)
			return
		}
	}(&ws)

	// Читатель
	go func(ws *WsConn) {
		for {
			_, message, err := ws.Conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Println("[error]", err)
				}
				break
			}

			ws.Reader <- message
		}

		select {
		case <-ws.Close:
		default:
			close(ws.Close)
		}
	}(&ws)

	// Писатель
	go func(ws *WsConn) {
		ticker := time.NewTicker(wsPingPeriod)
		defer func() {
			ticker.Stop()
			select {
			case <-ws.Close:
			default:
				close(ws.Close)
			}
		}()

		for {
			select {
			case message, ok := <-ws.Writer:
				ws.Conn.SetWriteDeadline(time.Now().Add(wsWriteTimeout))
				if !ok {
					return
				}

				w, err := ws.Conn.NextWriter(websocket.TextMessage)
				if err != nil {
					log.Println("[error]", err)
					return
				}
				w.Write(message)

				if err := w.Close(); err != nil {
					log.Println("[error]", err)
					return
				}
			case <-ticker.C:
				ws.Conn.SetWriteDeadline(time.Now().Add(wsWriteTimeout))
				if err := ws.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {

					return
				}
			}
		}
	}(&ws)

	go params.WsRoute(r, &ws)
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

	if len(wo.Ans.CspMap) == 0 {
		// Добавляем csp
		if params.Csp != "" {
			wo.W.Header().Add("Content-Security-Policy", params.Csp)
		}
	} else {
		csp := getCsp(wo.Ans.CspMap)
		if csp != "" {
			wo.W.Header().Add("Content-Security-Policy", csp)
		}
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
