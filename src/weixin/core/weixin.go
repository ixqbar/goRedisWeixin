package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/tidwall/gjson"
	"io/ioutil"
	"sync"
	"time"
	"weixin/common"
)

type WValues struct {
	expireAt time.Time
	value    string
}

type WResponse struct {
	Errcode     int    `json:"errcode"`
	Errmsg      string `json:"errmsg"`
	Ticket      string `json:"ticket"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

type Weixin struct {
	sync.Mutex
	tokens  map[string]*WValues
	tickets map[string]*WValues
}

var wx *Weixin

func init() {
	wx = &Weixin{
		tokens:  make(map[string]*WValues, 0),
		tickets: make(map[string]*WValues, 0),
	}
}

func GetToken(name string, cacheFirst bool) (string, error) {
	appId := common.Config.IniCfg.Section(name).Key("app_id").String()
	appSecret := common.Config.IniCfg.Section(name).Key("app_secret").String()

	if len(appId) == 0 || len(appSecret) == 0 {
		return "", errors.New("ERR not found match gzh config")
	}

	return wx.getToken(appId, appSecret, cacheFirst, true)
}

func GetTicket(name string, cacheFirst bool) (string, error) {
	appId := common.Config.IniCfg.Section(name).Key("app_id").String()
	appSecret := common.Config.IniCfg.Section(name).Key("app_secret").String()

	if len(appId) == 0 || len(appSecret) == 0 {
		return "", errors.New("ERR not found match gzh config")
	}

	return wx.getTicket(appId, appSecret, cacheFirst)
}

func (w *Weixin) getToken(appId, appSecret string, cacheFirst, autoLock bool) (string, error) {
	if autoLock {
		w.Lock()
		defer w.Unlock()
	}

	if cacheFirst {
		if v, ok := w.tokens[appId]; ok && v.expireAt.After(time.Now()) {
			return v.value, nil
		}
	}

	/**
	https://developers.weixin.qq.com/doc/offiaccount/Basic_Information/Get_access_token.html

	*/
	tokenApiUrl := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s", appId, appSecret)

	res, err := Get(tokenApiUrl).Bytes()
	if err != nil {
		common.Logger.Printf("request weixin token api fail api=%s", tokenApiUrl)
		return "", errors.New("request weixin token api fail")
	}

	var wRes WResponse

	err = json.Unmarshal(res, &wRes)
	if err != nil || len(wRes.AccessToken) == 0 {
		common.Logger.Printf("parse weixin token api response fail api=%s, response=%s", tokenApiUrl, string(res))
		return "", errors.New("parse weixin token api response fail")
	}

	w.tokens[appId] = &WValues{
		expireAt: time.Now().Add(time.Second * time.Duration(wRes.ExpiresIn-10)),
		value:    wRes.AccessToken,
	}

	w.save()

	common.Logger.Printf("refresh weixin token success appId=%s,token=%s,expireAt=%s", appId, wRes.AccessToken, w.tokens[appId].expireAt.Format("2006-01-02 15:04:05"))

	return wRes.AccessToken, nil
}

func (w *Weixin) getTicket(appId, appSecret string, cacheFirst bool) (string, error) {
	w.Lock()
	defer w.Unlock()

	if cacheFirst {
		if v, ok := w.tickets[appId]; ok && v.expireAt.After(time.Now()) {
			return v.value, nil
		}
	}

	accessToken, err := w.getToken(appId, appSecret, cacheFirst, false)
	if err != nil {
		return "", err
	}

	/**
	https://developers.weixin.qq.com/doc/offiaccount/OA_Web_Apps/JS-SDK.html
	*/
	tokenApiUrl := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/ticket/getticket?access_token=%s&type=jsapi", accessToken)

	res, err := Get(tokenApiUrl).Bytes()
	if err != nil {
		common.Logger.Printf("request weixin ticket api fail api=%s", tokenApiUrl)
		return "", errors.New("request weixin ticket api fail")
	}

	var wRes WResponse

	err = json.Unmarshal(res, &wRes)
	if err != nil || len(wRes.Ticket) == 0 {
		common.Logger.Printf("parse weixin ticket api response fail api=%s, response=%s", tokenApiUrl, string(res))
		return "", errors.New("parse weixin ticket api response fail")
	}

	w.tickets[appId] = &WValues{
		expireAt: time.Now().Add(time.Second * time.Duration(wRes.ExpiresIn-10)),
		value:    wRes.Ticket,
	}

	w.save()

	common.Logger.Printf("refresh weixin ticket success appId=%s,ticket=%s,expireAt=%s", appId, wRes.Ticket, w.tokens[appId].expireAt.Format("2006-01-02 15:04:05"))

	return wRes.Ticket, nil
}

func (w *Weixin) LoadData() {
	w.Lock()
	defer w.Unlock()

	if len(common.Config.DataFile) == 0 {
		common.Logger.Print("not found data file")
		return
	}

	jsonContent, err := ioutil.ReadFile(common.Config.DataFile)
	if err != nil {
		common.Logger.Printf("read data file fail %v", err)
		return
	}

	tokens := gjson.Get(string(jsonContent), "tokens")
	tokens.ForEach(func(key, value gjson.Result) bool {
		expireAt := time.Unix(value.Get("expireAt").Int(), 0)
		if expireAt.Before(time.Now()) {
			return true
		}

		common.Logger.Printf("iterate token,appId=%s,token=%s,expireAt=%s", key.String(), value.Get("token").String(), expireAt.String())

		wx.tokens[key.String()] = &WValues{
			expireAt: expireAt,
			value:    value.Get("token").String(),
		}

		return true
	})

	tickets := gjson.Get(string(jsonContent), "tickets")
	tickets.ForEach(func(key, value gjson.Result) bool {
		expireAt := time.Unix(value.Get("expireAt").Int(), 0)
		if expireAt.Before(time.Now()) {
			return true
		}

		common.Logger.Printf("iterate ticket,appId=%s,ticket=%s,expireAt=%s", key.String(), value.Get("ticket").String(), expireAt.String())

		wx.tickets[key.String()] = &WValues{
			expireAt: expireAt,
			value:    value.Get("ticket").String(),
		}

		return true
	})
}

func (w *Weixin) save() {
	var jsonContent bytes.Buffer

	jsonContent.WriteString("{\"tokens\":{")

	tokenKeyIsFirst := true
	for k, v := range w.tokens {
		if v.expireAt.Before(time.Now()) {
			continue
		}
		if tokenKeyIsFirst {
			tokenKeyIsFirst = false
			jsonContent.WriteString(fmt.Sprintf("\"%s\":{\"expireAt\":%d,\"token\":\"%s\"}", k, v.expireAt.Unix(), v.value))
		} else {
			jsonContent.WriteString(fmt.Sprintf(",\"%s\":{\"expireAt\":%d,\"token\":\"%s\"}", k, v.expireAt.Unix(), v.value))
		}
	}
	jsonContent.WriteString("}")

	jsonContent.WriteString(",\"tickets\":{")

	ticketKeyIsFirst := true
	for k, v := range w.tickets {
		if v.expireAt.Before(time.Now()) {
			continue
		}
		if ticketKeyIsFirst {
			ticketKeyIsFirst = false
			jsonContent.WriteString(fmt.Sprintf("\"%s\":{\"expireAt\":%d,\"ticket\":\"%s\"}", k, v.expireAt.Unix(), v.value))
		} else {
			jsonContent.WriteString(fmt.Sprintf(",\"%s\":{\"expireAt\":%d,\"ticket\":\"%s\"}", k, v.expireAt.Unix(), v.value))
		}
	}
	jsonContent.WriteString("}")

	jsonContent.WriteString("}")

	ioutil.WriteFile(common.Config.DataFile, jsonContent.Bytes(), 0644)
}
