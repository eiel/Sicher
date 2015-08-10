package sicher

import (
    "fmt"
    "net/http"
    "net/url"
    "appengine"
    "appengine/urlfetch"
    "appengine/taskqueue"
    "io/ioutil"
)

func init() {
    http.HandleFunc("/", handler)
    http.HandleFunc("/checks", checksHandler)
    http.HandleFunc("/notification/slack", slackHandler)
}

func handler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprint(w, "Hello, world!")
}

func checksHandler(w http.ResponseWriter, r *http.Request) {
    c := appengine.NewContext(r)
    client := urlfetch.Client(c)
    myurl := "http://eiel.info/"
    resp, err := client.Head(myurl)
    if err != nil {
        c.Debugf("Fail",)
       fmt.Fprint(w, err.Error())
       // TODO datastroeに記録する
       // TODO 通知する
       // TODO 状態を失敗に変更
    } else {
        c.Debugf("Success",)
        fmt.Fprint(w, resp.Status)
       // TODO datastroeに記録する
       // TODO 高度なチェックを起動
       // TODO 状態が失敗なら成功になったことを通知
       // TODO 状態を成功に変更
    }
    t := taskqueue.NewPOSTTask("/notification/slack",
      map[string][]string{"url": {myurl}})
    if _, err := taskqueue.Add(c, t, ""); err != nil {
       http.Error(w, err.Error(), http.StatusInternalServerError)
       return
    }
}

func slackHandler(w http.ResponseWriter, r *http.Request) {
    myurl := r.FormValue("url")

    c := appengine.NewContext(r)
    token := "xoxp-XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"
    slackUrl := "https://slack.com/api/chat.postMessage"
    values := url.Values{
      "token":{token},
      "username":{"sicher"},
      "text":{"hping:" + myurl + " fail."},
      "channel":{"#general"},
    }
   client := urlfetch.Client(c)
   resp ,err  := client.PostForm(slackUrl, values)
    if err != nil {
         c.Infof("slackHandler: {err2.Error()}",)
    } else {
       contents, _ := ioutil.ReadAll(resp.Body)
       fmt.Fprint(w, string(contents))
    }
}
