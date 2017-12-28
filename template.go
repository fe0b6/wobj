package wobj

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"time"
)

// Проверяем JS запрос или нет
func (wo *Obj) isJs() (ok bool) {
	if wo.R.FormValue("js") == "1" {
		ok = true
	}
	return
}

// Tmpl преобразуем объект в json и шаблонизируем
func (wo *Obj) Tmpl() (str string, err error) {

	var o map[string]interface{}
	// Собираем данные в нужный вид
	if len(wo.Ans.Path) > 0 {
		o = map[string]interface{}{wo.Ans.Path[len(wo.Ans.Path)-1]: wo.Ans.Data}
		for i := len(wo.Ans.Path) - 2; i >= 0; i-- {
			o = map[string]interface{}{wo.Ans.Path[i]: o}
		}
	} else {
		o = map[string]interface{}{"data": wo.Ans.Data}
	}

	// Если это не js - добавляем контент
	if !wo.isJs() {
		o = map[string]interface{}{"content": o}
	} else {
		o["_tmpl"] = "main"
	}

	// Дополнительные параметры
	if wo.Ans.Meta.Title != "" {
		o["meta"] = wo.Ans.Meta
	}
	o["now_year"] = time.Now().Year()

	// Добавляем другие нужные значения
	if wo.AppendFunc != nil {
		o = wo.AppendFunc(wo, o)
	}

	// Делаем json
	var js []byte
	js, err = json.Marshal(o)
	if err != nil {
		log.Println("[error]", err)
		return
	}

	// Если это js - не шаблонизируем
	if wo.isJs() {
		str = string(js)
		return
	}

	return wo.objToHTML(js)
}

// Преобразуем объект в html
func (wo *Obj) objToHTML(js []byte) (str string, err error) {

	if wo.Debug {
		log.Println("[debug]", string(js))
	}

	if len(js) > params.MaxArgLeg {
		var f *os.File
		if f, err = ioutil.TempFile("/tmp/", "yate_tmpl_"); err != nil {
			return
		}
		defer os.Remove(f.Name())

		if _, err = f.Write(js); err != nil {
			return
		}

		if err = f.Close(); err != nil {
			return
		}

		js, err = json.Marshal(map[string]interface{}{"__filename": f.Name()})
		if err != nil {
			return
		}
	}

	cmd := exec.Command(params.NodeScript, params.YateScript, string(js))
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Println("[error]", string(out))
		log.Println("[error]", len(js))
		return
	}

	str = string(out)

	return
}
