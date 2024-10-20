package main

import (
    "bytes"
    "fmt"
    "encoding/json"
    "net/http"
    "io/ioutil"
    "time"
    "strconv"
    "strings"
)

type Pagination struct {
	Page      int `json:"page"`
	PageCount int `json:"pageCount"`
}

type Meta struct {
	Pagination Pagination `json:"pagination"`
}

type Strapi struct {
    Endpoint string
    Token    string
}

type StrapiResponse interface {
    PageCount() int
    Add(data StrapiResponse)
    NewInstance() StrapiResponse
    ResourceName() string
}

type StrapiType interface {
   GetId() int
   ResourceName() string
   AsWriter() interface{}
   AsNullableWriter() interface{}
}

type StrapiTypePtr interface {
   PtrResourceName() string
}

type StrapiWriterAttrs interface {
    WriterResourceName() string
}

func IntPtr(value int) *int {
    return &value
}

func (s *Strapi) GetInto(route string, tgt interface{}) error {
    url := s.Endpoint + route
    req, err := http.NewRequest(http.MethodGet, url, nil)
    if err != nil {
        return err
    }
    req.Header.Set("Authorization", "Bearer " + s.Token)
    client := http.Client{Timeout: 30 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    if resp.StatusCode != 200 {
        defer resp.Body.Close()
        response, _ := ioutil.ReadAll(resp.Body)
        return fmt.Errorf("Status code %v: %v", resp.StatusCode, string(response))
    }
    defer resp.Body.Close()
    response, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return err
    }
    err = json.Unmarshal(response, tgt)
    return err
}

func AddFilters(route string, filters... string) string {
     pidx := 0
    for _, f := range filters {
        if f == "*" {
            route += "&populate=*"
            pidx += 1
            continue
        }
        hit := false
        for k, v := range map[string]string {">=": "$ge", "<=": "$le", ">":"$gt", "<":"$lt", "=":"$eq"} {
            if strings.Contains(f, k) {
                r := "&filters"
                kv := strings.Split(f, k)
                for _, item := range strings.Split(kv[0], ".") {
                  r += "["+item+"]"
                }
                r += "["+v+"]=" + kv[1]
                route += r
                hit = true
                break
            }
        } 
        if hit {
            continue
        }
        r := "&populate" // [" + strconv.Itoa(pidx) + "]"
        pidx += 1
        items := strings.Split(f, ".")
        if len(items) == 1 {
            r += "=" + f
            route += r
            continue
        }
        for _, k := range items[0:len(items)-1] {
            r += "[" + k + "]"
        }
        last := items[len(items)-1]
        if last[len(last)-1] == '*' {
            r += "[populate]=" + last[0:len(last)-1]
        } else {
            r += "[fields]=" + last
        }
        route += r
    }
    return route
}

func (s *Strapi) Get(tgt StrapiTypePtr, id int, filters... string) error {
    route := "/" + tgt.PtrResourceName() + "/" + strconv.Itoa(id) + "?";
    route = AddFilters(route, filters...)
    fmt.Println(route)
    return s.GetInto(route, tgt)
}

func (s *Strapi) List(tgt StrapiResponse, filters... string) error {
    route := "/" + tgt.ResourceName() + "?";
    route = AddFilters(route, filters...)
    fmt.Println(route)
    return s.GetIntoPaginated(route, tgt)
}

func (s *Strapi) GetIntoPaginated(route string, tgt StrapiResponse) error {
    page := 1
    for {
        proute := route + "&pagination[page]=" + strconv.Itoa(page)
        tmp := tgt.NewInstance()
        err := s.GetInto(proute, tmp)
        if err != nil {
            return err
        }
        tgt.Add(tmp)
        if tmp.PageCount() <= page {
            break
        }
        page += 1
    }
    return nil
}

func (s *Strapi) Update(v StrapiType) error {
    id := v.GetId()
    w := v.AsWriter()
    resource := v.ResourceName()
    return s.UpdateFrom(resource, id, w)
}

func (s *Strapi) UpdateNullable(v StrapiType) error {
    id := v.GetId()
    w := v.AsNullableWriter()
    resource := v.ResourceName()
    return s.UpdateFrom(resource, id, w)
}

func (s *Strapi) UpdateFromWriterAttrs(id int, v StrapiWriterAttrs) error {
    resource := v.WriterResourceName()
    return s.UpdateFrom(resource, id, v)
}

func (s *Strapi) UpdateFrom(resource string, id int, w interface{}) error {
    body := make(map[string]interface{})
    body["data"] = w
    j, err := json.Marshal(body)
    fmt.Printf("%v\n", string(j))
    if err != nil {
        return err
    }
    url := s.Endpoint + "/" + resource + "/" + strconv.Itoa(id)
    req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(j))
    if err != nil {
        return err
    }
    req.Header.Set("Authorization", "Bearer " + s.Token)
    req.Header.Set("Content-Type", "application/json")
    client := http.Client{Timeout: 30 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    if resp.StatusCode != 200 {
        defer resp.Body.Close()
        response, _ := ioutil.ReadAll(resp.Body)
        return fmt.Errorf("Status code %v: %v", resp.StatusCode, string(response))
    }
    defer resp.Body.Close()
    return nil
}

func (s *Strapi) Add(v StrapiWriterAttrs) (int, error) {
    return s.AddFrom(v.WriterResourceName(), v)
}

func (s *Strapi) AddFrom(resource string, w interface{}) (int, error) {
    body := make(map[string]interface{})
    body["data"] = w
    j, err := json.Marshal(body)
    fmt.Printf("%v\n", string(j))
    if err != nil {
        return -1, err
    }
    url := s.Endpoint + "/" + resource
    req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(j))
    if err != nil {
        return -1, err
    }
    req.Header.Set("Authorization", "Bearer " + s.Token)
    req.Header.Set("Content-Type", "application/json")
    client := http.Client{Timeout: 30 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        return -1, err
    }
    if resp.StatusCode != 200 {
        defer resp.Body.Close()
        response, _ := ioutil.ReadAll(resp.Body)
        return -1, fmt.Errorf("Status code %v: %v", resp.StatusCode, string(response))
    }
    defer resp.Body.Close()
    response, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return -1, err
    }
    fmt.Printf("ADD: %v\n", string(response))
    jd := make(map[string]interface{})
    err = json.Unmarshal(response, &jd)
    if err != nil {
        fmt.Printf("no unmarshal %v\n", err)
        return 0, nil
    }
    jdat, ok := jd["data"]
    if !ok {
        fmt.Println("no data");
        return 0, nil
    }
    jdm, ok := jdat.(map[string]interface{})
    if !ok {
        fmt.Println("data not a map");
        return 0, nil
    }
    idi, ok := jdm["id"]
    if !ok {
        fmt.Println("map not id");
        return 0, nil
    }
    idf, ok := idi.(float64)
    if !ok {
        fmt.Println("id not float");
        return 0, nil
    }
    return int(idf), nil
}

func (s *Strapi) Delete(v StrapiType) error {
    id := v.GetId()
    resource := v.ResourceName()
    return s.DeleteResource(resource, id)
}

func (s *Strapi) DeleteResource(resource string, id int) error {
    url := s.Endpoint + "/" + resource + "/" + strconv.Itoa(id)
    req, err := http.NewRequest(http.MethodDelete, url, nil)
    if err != nil {
        return err
    }
    req.Header.Set("Authorization", "Bearer " + s.Token)
    client := http.Client{Timeout: 30 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    if resp.StatusCode != 200 {
        defer resp.Body.Close()
        response, _ := ioutil.ReadAll(resp.Body)
        return fmt.Errorf("Status code %v: %v", resp.StatusCode, string(response))
    }
    defer resp.Body.Close()
    return nil
}