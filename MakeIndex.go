package main

import (
	"encoding/xml"
	"fmt"
	"strings"
	"net/http"
	"io/ioutil"
	"encoding/json"
)

type Capabilities struct {
	ID   string `json:"ID"`
	Author string `json:"Author"`
	Title string `json:"Title"`
}

type Metadata struct {
	Author []string `xml:"author"`
	Title   []string `xml:"title"`
}

type Inventory struct {
	Files []string `xml:"a"`
}

func DelFrSlice(strslice []string, feature string) []string{
	var result []string
	for i := range strslice {
		if strings.Contains(strslice[i], feature) {
			result = append(result, strslice[i])
		}
		}
		return result
}

func main() {

	data, err := getContent("http://localhost:8080/static/OPP/")
	if err != nil {
		fmt.Println("I felt a great disturbance in the Force, as if millions of requests suddenly cried out in terror and were suddenly silenced.")
	}
	Files := ExtractInventory(data)
	Files = DelFrSlice(Files, ".xml")
	var capabilities []Capabilities

	for i := range Files {
		http_req := "http://localhost:8080/static/OPP/" + Files[i]
	data, err = getContent(http_req)
	if err != nil {
		fmt.Println("I felt a great disturbance in the Force, as if millions of requests suddenly cried out in terror and were suddenly silenced.")
	}
	capabilities = BuildCapabilities(data, strings.Split(Files[i], ".xml")[0],capabilities)
}
capabilitiesJson, _ := json.Marshal(capabilities)
fmt.Println(string(capabilitiesJson))
}

func getContent(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("GET error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Status error: %v", resp.StatusCode)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Read body: %v", err)
	}

	return data, nil
}

func ExtractInventory(xmlbyte []byte) []string {
	var l Inventory
	decoder := xml.NewDecoder(strings.NewReader(string(xmlbyte)))
	for {
		// Read tokens from the XML document in a stream.
		token, _ := decoder.Token()
		if token == nil {
			break
		}
		switch Element := token.(type) {
		case xml.StartElement:
			if Element.Name.Local == "pre" {
				err := decoder.DecodeElement(&l, &Element)
				if err != nil {
					fmt.Println(err)
				}
				return l.Files
						}
		}
}
return []string{"Parser failed"}
}

func BuildCapabilities(xmlbyte []byte, urn string, capabilities []Capabilities) []Capabilities {
	var l Metadata
	decoder := xml.NewDecoder(strings.NewReader(string(xmlbyte)))
	for {
		// Read tokens from the XML document in a stream.
		token, _ := decoder.Token()
		if token == nil {
			break
		}
		switch Element := token.(type) {
		case xml.StartElement:
			if Element.Name.Local == "titleStmt" {
				err := decoder.DecodeElement(&l, &Element)
				if err != nil {
					fmt.Println(err)
				}
				capabilities = append(capabilities, Capabilities{
    			ID:   urn,
    			Author: strings.Join(l.Author, ","),
					Title: strings.Join(l.Title, ","),
  		})
				return capabilities
						}
		}
}
return capabilities
}
