package main

import (
	"github.com/aebruno/nwalgo"
	// "strconv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/gorilla/mux"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Word struct {
	Appearance string
	Id         int
}

type Node struct {
	URN      []string `json:"urn"`
	Text     []string `json:"text,omitempty"`
	Previous []string `json:"previous"`
	Next     []string `json:"next"`
	Index    int      `json:"sequence"`
}

type NodeResponse struct {
	RequestUrn []string `json:"requestUrn"`
	Status     string   `json:"status"`
	Service    string   `json:"service"`
	Message    string   `json:"message,omitempty"`
	URN        []string `json:"urns,omitempty"`
	Nodes      []Node   `json:""`
}

type Capabilities struct {
	ID     string `json:"ID"`
	Author string `json:"Author"`
	Title  string `json:"Title"`
}

type Metadata struct {
	Author []string `xml:"author"`
	Title  []string `xml:"title"`
}

type Inventory struct {
	Files []string `xml:"a"`
}

type ServerConfig struct {
	Host      string `json:"host"`
	Port      string `json:"port"`
	XMLSource string `json:"xml_source"`
	CEXSource string `json:"cex_source"`
}

type CTSXMLPage struct {
	Title         template.HTML
	Passage       template.HTML
	AlignmentDivs template.HTML
}

type ParsedCTS struct {
	Title   string
	Author  string
	Passage string
}

type CTSParams struct {
	Sourcetext, StartID, EndID string
}

func maxfloat(floatslice []float64) int {
	max := floatslice[0]
	maxindex := 0
	for i, value := range floatslice {
		if value > max {
			max = value
			maxindex = i
		}
	}
	return maxindex
}

func nwastrings(collection []string) string {
	start := `	<div class="tbody">
		<form class="tr">
`
	start2 := `	<div class="thead">
		<div class="tr">
`
	end := `<div class="td action"><button type="button" onclick="edit(this);">edit</button></div>
		</form></div>`
	end2 := `</div>
	</div>`

	button := `<button type="button" onclick="edit(this);">edit</button>`

	var output string

	for i := range collection {
		collection[i] = strings.ToLower(collection[i])
	}
	basestring := strings.Fields(collection[0])
	var basetext []Word
	var comparetext []Word
	for i := range basestring {
		basetext = append(basetext, Word{Appearance: basestring[i], Id: i + 1})
	}
	output = start + button
	for i := range basetext {
		output = output + "<div class=\"td\">" + basetext[i].Appearance + "</div>" + "\n"
	}
	output = output + end
	output = output + start2 + "<div class=\"td\">Edit</div>" + "\n"
	for i := range basetext {
		output = output + "<div class=\"td\">" + basetext[i].Appearance + "</div>" + "\n"
	}
	output = output + end2

	for i := 0; i < len(collection)-1; i++ {
		comparestring := strings.Fields(collection[i+1])
		comparetext = []Word{}
		for i := range comparestring {
			comparetext = append(comparetext, Word{Appearance: comparestring[i]})
		}
		aln1, aln2, score := nwalgo.Align(collection[0], collection[i+1], 1, -1, -1)
		aligned1 := strings.Fields(aln1)
		aligned2 := strings.Fields(aln2)
		var score_range []float64
		var index int
		for j := range aligned1 {
			score_range = []float64{}
			for i, _ := range aligned2 {
				aln1, aln2, score = nwalgo.Align(aligned1[j], aligned2[i], 1, -1, -1)
				var penalty float64
				switch {
				case i > j:
					penalty = float64((i - j)) / 2.0
				case i < j:
					penalty = float64((j - i)) / 2.0
				default:
					penalty = 0
				}
				var f float64 = (float64(score) - penalty) / float64(len(aln1))
				score_range = append(score_range, f)
			}
			index = maxfloat(score_range)
			comparetext[index].Id = j + 1
		}
		output = output + start + button
		for i := range comparetext {
			output = output + "<div class=\"td\">" + comparetext[i].Appearance + "</div>" + "\n"
		}
		output = output + end
	}
	return output
}

func DelFrSlice(strslice []string, feature string) []string {
	var result []string
	for i := range strslice {
		if strings.Contains(strslice[i], feature) {
			result = append(result, strslice[i])
		}
	}
	return result
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
					ID:     urn,
					Author: strings.Join(l.Author, ","),
					Title:  strings.Join(l.Title, ","),
				})
				return capabilities
			}
		}
	}
	return capabilities
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

func LoadConfiguration(file string) ServerConfig {
	var config ServerConfig
	configFile, err := os.Open(file)
	defer configFile.Close()
	if err != nil {
		fmt.Println(err.Error())
	}
	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&config)
	return config
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

func ParseCTS(p CTSParams) ParsedCTS {
	type REFSECTION struct {
		Value string `xml:"replacementPattern,attr"`
	}

	type REFDATA struct {
		Value []REFSECTION `xml:"cRefPattern"`
	}

	type Chunk struct {
		Text string `xml:",chardata"`
		ID   string `xml:"n,attr"`
	}

	type TitleData struct {
		Title  []string `xml:"title"`
		Author []string `xml:"author"`
	}

	type HEADERDATA struct {
		Refs   REFDATA   `xml:"teiHeader>encodingDesc>refsDecl"`
		Titles TitleData `xml:"teiHeader>fileDesc>titleStmt"`
	}

	re_inside_whtsp := regexp.MustCompile(`[\s\p{Zs}]{2,}`)
	var result string
	var identifiers []string
	var text_content []string
	startid := p.StartID
	endid := p.EndID

	confvar := LoadConfiguration("config.json")

	input_file := confvar.XMLSource + p.Sourcetext

	data, err := getContent(input_file)
	if err != nil {
		return ParsedCTS{Passage: "I felt a great disturbance in the Force, as if millions of requests suddenly cried out in terror and were suddenly silenced."}
	}

	var vr HEADERDATA

	err2 := xml.Unmarshal([]byte(data), &vr)
	if err2 != nil {
		fmt.Printf("error: %vr", err)
		return ParsedCTS{Passage: "I felt a great disturbance in the Force, as if millions of requests suddenly cried out in terror and were suddenly silenced."}
	}
	RefString := vr.Refs.Value[0].Value
	RefString = strings.Replace(RefString, "#xpath(/tei:TEI/", "", -1)
	RefString = strings.Replace(RefString, "tei:", "", -1)
	RefString = strings.Replace(RefString, "text/body/div/", "", -1)
	RefString = strings.Replace(RefString, ")", "", -1)
	reg, err := regexp.Compile("[0-9]")
	if err != nil {
		log.Fatal(err)
	}
	RefString = reg.ReplaceAllString(RefString, "")
	RefString = strings.Replace(RefString, "[@n='$']", "", -1)
	ValidRefs := strings.Split(RefString, "/")
	ValidRefs = delete_empty(ValidRefs)
	var xmltag string

	switch {
	case len(ValidRefs) > 3:
		// still to do
		switch {
		case slistrContains(ValidRefs, "l"):
			xmltag = "l"
		default:
			xmltag = "p"
		}
	case len(ValidRefs) > 2:
		type NODE2 struct {
			ID    string `xml:"n,attr"`
			Inner string `xml:",innerxml"`
		}

		type NODE1 struct {
			ID     string  `xml:"n,attr"`
			NODE2s []NODE2 `xml:"div"`
		}

		type TEXTDATA struct {
			NODE1s []NODE1 `xml:"text>body>div>div"`
		}
		switch {
		case slistrContains(ValidRefs, "l"):
			xmltag = "l"
		default:
			xmltag = "p"
		}
		var vv TEXTDATA

		err2 := xml.Unmarshal([]byte(data), &vv)
		if err2 != nil {
			fmt.Printf("error: %vv", err)
			return ParsedCTS{Passage: "I felt a great disturbance in the Force, as if millions of requests suddenly cried out in terror and were suddenly silenced."}
		}

		var stringset []string

		for i := 0; i < len(vv.NODE1s); i++ {
			for ii := 0; ii < len(vv.NODE1s[i].NODE2s); ii++ {
				var l Chunk

				decoder := xml.NewDecoder(strings.NewReader(string(vv.NODE1s[i].NODE2s[ii].Inner)))
				var ind_id string
				for {
					// Read tokens from the XML document in a stream.
					token, _ := decoder.Token()
					if token == nil {
						break
					}
					switch Element := token.(type) {
					case xml.StartElement:
						if Element.Name.Local == xmltag {
							err := decoder.DecodeElement(&l, &Element)
							if err != nil {
								fmt.Println(err)
							}
							text_content = append(text_content, strings.TrimSpace(l.Text))
						}
					}
				}
				decoder = xml.NewDecoder(strings.NewReader(string(vv.NODE1s[i].NODE2s[ii].Inner)))
				for {
					// Read tokens from the XML document in a stream.
					token, _ := decoder.Token()
					if token == nil {
						break
					}
					switch Element := token.(type) {
					case xml.StartElement:
						if Element.Name.Local == "div" {
							err := decoder.DecodeElement(&l, &Element)
							if err != nil {
								fmt.Println(err)
							}
							stringset = []string{vv.NODE1s[i].ID, vv.NODE1s[i].NODE2s[ii].ID, l.ID}
							ind_id = strings.Join(stringset, ".")
							identifiers = append(identifiers, ind_id)
						}
					}
				}
			}
		}
	case len(ValidRefs) > 1:
		type NODE1 struct {
			ID    string `xml:"n,attr"`
			Inner string `xml:",innerxml"`
		}

		type TEXTDATA struct {
			NODE1s []NODE1 `xml:"text>body>div>div"`
		}
		switch {
		case slistrContains(ValidRefs, "l"):
			xmltag = "l"
		default:
			xmltag = "p"
		}
		var vv TEXTDATA

		err2 := xml.Unmarshal([]byte(data), &vv)
		if err2 != nil {
			fmt.Printf("error: %vv", err)
			return ParsedCTS{Passage: "I felt a great disturbance in the Force, as if millions of requests suddenly cried out in terror and were suddenly silenced."}
		}

		var stringset []string

		for i := 0; i < len(vv.NODE1s); i++ {
			var l Chunk

			decoder := xml.NewDecoder(strings.NewReader(string(vv.NODE1s[i].Inner)))
			var ind_id string
			for {
				// Read tokens from the XML document in a stream.
				token, _ := decoder.Token()
				if token == nil {
					break
				}
				switch Element := token.(type) {
				case xml.StartElement:
					if Element.Name.Local == xmltag {
						err := decoder.DecodeElement(&l, &Element)
						if err != nil {
							fmt.Println(err)
						}
						stringset = []string{vv.NODE1s[i].ID, l.ID}
						ind_id = strings.Join(stringset, ".")
						identifiers = append(identifiers, ind_id)
						text_content = append(text_content, strings.TrimSpace(l.Text))
					}
				}
			}
		}
	default:
		type NODE1 struct {
			ID    string `xml:"n,attr"`
			Inner string `xml:",innerxml"`
		}

		type TEXTDATA struct {
			NODE1s []NODE1 `xml:"text>body>div"`
		}

		switch {
		case slistrContains(ValidRefs, "l"):
			xmltag = "l"
		default:
			xmltag = "p"
		}
		var vv TEXTDATA

		err2 := xml.Unmarshal([]byte(data), &vv)
		if err2 != nil {
			fmt.Printf("error: %vv", err)
			return ParsedCTS{Passage: "I felt a great disturbance in the Force, as if millions of requests suddenly cried out in terror and were suddenly silenced."}
		}

		for i := 0; i < len(vv.NODE1s); i++ {
			var l Chunk

			decoder := xml.NewDecoder(strings.NewReader(string(vv.NODE1s[i].Inner)))
			var ind_id string
			for {
				// Read tokens from the XML document in a stream.
				token, _ := decoder.Token()
				if token == nil {
					break
				}
				switch Element := token.(type) {
				case xml.StartElement:
					if Element.Name.Local == xmltag {
						err := decoder.DecodeElement(&l, &Element)
						if err != nil {
							fmt.Println(err)
						}
						ind_id = l.ID
						identifiers = append(identifiers, ind_id)
						text_content = append(text_content, strings.TrimSpace(l.Text))
					}
				}
			}
		}
	}

	switch {
	case startid != "" && endid != "":
		var index1 int
		var index2 int
		var startreplace, endreplace string
		if strings.Contains(startid, "@") {
			startreplace = strings.Split(startid, "@")[1]
			startid = strings.Split(startid, "@")[0]
			index1 = finder(identifiers, startid)
			text_content[index1] = startreplace + after(text_content[index1], startreplace)
		}
		if strings.Contains(endid, "@") {
			endreplace = strings.Split(endid, "@")[1]
			endid = strings.Split(endid, "@")[0]
			index2 = finder(identifiers, endid)
			text_content[index2] = before(text_content[index2], endreplace) + endreplace
		}
		index1 = finder(identifiers, startid)
		index2 = finder(identifiers, endid) + 1
		identifiers = identifiers[index1:index2]
		if startreplace != "" {
			identifiers[0] = identifiers[0] + "@" + startreplace
		}
		if endreplace != "" {
			identifiers[len(identifiers)-1] = identifiers[len(identifiers)-1] + "@" + endreplace
		}
		text_content = text_content[index1:index2]
		for i, _ := range identifiers {
			text_content[i] = re_inside_whtsp.ReplaceAllString(text_content[i], " ")
			text_content[i] = strings.Replace(text_content[i], "\n", "<br>", -1)
			text_content[i] = "<div n=\"" + identifiers[i] + "\">" + text_content[i] + "</div>"
		}
		result = strings.Join(text_content, "</br>")
	case startid != "" && endid == "":
		var index1 int
		index1 = finder(identifiers, startid)
		text_content[index1] = re_inside_whtsp.ReplaceAllString(text_content[index1], " ")
		text_content[index1] = strings.Replace(text_content[index1], "\n", "<br>", -1)
		result = "<div n=\"" + identifiers[index1] + "\">" + text_content[index1] + "</div>"
	default:
		for i, _ := range identifiers {
			text_content[i] = re_inside_whtsp.ReplaceAllString(text_content[i], " ")
			text_content[i] = strings.Replace(text_content[i], "\n", "<br>", -1)
			text_content[i] = "<div n=\"" + identifiers[i] + "\">" + text_content[i] + "</div>"
		}
		result = strings.Join(text_content, "</br>")
	}
	result_struct := ParsedCTS{Title: vr.Titles.Title[0], Author: vr.Titles.Author[0], Passage: result}
	return result_struct
}

func main() {
	confvar := LoadConfiguration("./config.json")
	serverIP := confvar.Port
	router := mux.NewRouter().StrictSlash(true)
	s := http.StripPrefix("/static/", http.FileServer(http.Dir("./static/")))
	router.PathPrefix("/static/").Handler(s)
	router.HandleFunc("/cts", CTSIndex)
	router.HandleFunc("/GetCapabilities", GetCapabilities)
	router.HandleFunc("/cts/full/{sourcetext}/", CTSShowWork)
	router.HandleFunc("/cts/chunk/{sourcetext}:{ctsID}", CTSShow)
	router.HandleFunc("/cts/range/{sourcetext}:{ctsID}-{ctsID2}", CTSShowRange)
	router.HandleFunc("/cex/nwa/{urns}", NWAcex)
	router.HandleFunc("/{key}", serveTemplate)
	log.Println("Listening at" + serverIP + "...")
	log.Fatal(http.ListenAndServe(serverIP, router))
}

func NWAcex(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	urns := strings.Split(vars["urns"], "+")
	var nodelist NodeResponse
	var lookatnodes []string
	for i := range urns {
		urlstring := "http://127.0.0.1:8080/nyaya/texts/" + urns[i]
		res, err := http.Get(urlstring)

		if err != nil {
			panic(err.Error())
		}
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			panic(err.Error())
		}
		err = json.Unmarshal(body, &nodelist)
		lookatnodes = append(lookatnodes, nodelist.Nodes[0].Text[0])
	}
	output := nwastrings(lookatnodes)
	alignment := template.HTML(output)
	p := &CTSXMLPage{AlignmentDivs: alignment}
	lp := filepath.Join("templates", "alignment.html")
	t, _ := template.ParseFiles(lp)
	t.Execute(w, p)
}

func GetCapabilities(w http.ResponseWriter, r *http.Request) {

	confvar := LoadConfiguration("config.json")

	data, err := getContent(confvar.XMLSource)
	if err != nil {
		fmt.Println("I felt a great disturbance in the Force, as if millions of requests suddenly cried out in terror and were suddenly silenced.")
	}
	Files := ExtractInventory(data)
	Files = DelFrSlice(Files, ".xml")
	var capabilities []Capabilities

	for i := range Files {
		http_req := "http://localhost:8000/static/OPP/" + Files[i]
		data, err = getContent(http_req)
		if err != nil {
			fmt.Println("I felt a great disturbance in the Force, as if millions of requests suddenly cried out in terror and were suddenly silenced.")
		}
		capabilities = BuildCapabilities(data, strings.Split(Files[i], ".xml")[0], capabilities)
	}
	capabilitiesJson, _ := json.Marshal(capabilities)
	fmt.Fprintln(w, string(capabilitiesJson))
}

func CTSIndex(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "CTS Index!")
}

func CTSShow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	citation := vars["ctsID"]
	sourcetext := strings.Join([]string{vars["sourcetext"], "xml"}, ".")
	result := ParseCTS(CTSParams{Sourcetext: sourcetext, StartID: citation})
	passage := template.HTML(result.Passage)
	title_field := template.HTML("Title: " + result.Title + "</br>" + "Author: " + result.Author)
	p := &CTSXMLPage{Title: title_field, Passage: passage}
	lp := filepath.Join("templates", "layout.html")
	t, _ := template.ParseFiles(lp)
	t.Execute(w, p)
}

func CTSShowWork(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	citation := vars["ctsID"]
	sourcetext := strings.Join([]string{vars["sourcetext"], "xml"}, ".")
	result := ParseCTS(CTSParams{Sourcetext: sourcetext, StartID: citation})
	passage := template.HTML(result.Passage)
	title_field := template.HTML("Title: " + result.Title + "</br>" + "Author: " + result.Author)
	p := &CTSXMLPage{Title: title_field, Passage: passage}
	lp := filepath.Join("templates", "layout.html")
	t, _ := template.ParseFiles(lp)
	t.Execute(w, p)
}

func CTSShowRange(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	citation := vars["ctsID"]
	citation2 := vars["ctsID2"]
	sourcetext := strings.Join([]string{vars["sourcetext"], "xml"}, ".")
	result := ParseCTS(CTSParams{Sourcetext: sourcetext, StartID: citation, EndID: citation2})
	passage := template.HTML(result.Passage)
	title_field := template.HTML("Title: " + result.Title + "</br>" + "Author: " + result.Author)
	p := &CTSXMLPage{Title: title_field, Passage: passage}
	lp := filepath.Join("templates", "layout.html")
	t, _ := template.ParseFiles(lp)
	t.Execute(w, p)
}

func serveTemplate(w http.ResponseWriter, r *http.Request) {
	lp := filepath.Join("templates", "layout.html")
	vars := mux.Vars(r)
	id := vars["key"]
	fp := filepath.Join("templates", filepath.Clean(id))

	// Return a 404 if the template doesn't exist
	info, err := os.Stat(fp)
	if err != nil {
		if os.IsNotExist(err) {
			http.NotFound(w, r)
			return
		}
	}

	// Return a 404 if the request is for a directory
	if info.IsDir() {
		http.NotFound(w, r)
		return
	}

	tmpl, err := template.ParseFiles(lp, fp)
	if err != nil {
		// Log the detailed error
		log.Println(err.Error())
		// Return a generic "Internal Server Error" message
		http.Error(w, http.StatusText(500), 500)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "layout", nil); err != nil {
		log.Println(err.Error())
		http.Error(w, http.StatusText(500), 500)
	}
}

func before(value string, a string) string {
	// Get substring before a string.
	pos := strings.Index(value, a)
	if pos == -1 {
		return ""
	}
	return value[0:pos]
}

func after(value string, a string) string {
	// Get substring after a string.
	pos := strings.LastIndex(value, a)
	if pos == -1 {
		return ""
	}
	adjustedPos := pos + len(a)
	if adjustedPos >= len(value) {
		return ""
	}
	return value[adjustedPos:len(value)]
}

func slistrContains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func finder(s []string, e string) int {
	for i, a := range s {
		if a == e {
			return i
		}
	}
	fmt.Println("Index not found.")
	return 0
}

func delete_empty(s []string) []string {
	var r []string
	for _, str := range s {
		if str != "" {
			r = append(r, str)
		}
	}
	return r
}
