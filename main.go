package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

var allOUIs []*OUIData

type OUIData struct {
	OUI                 string `json:"oui"`
	VendorName          string `json:"vendor_name"`
	VendorAlternateName string `json:"vendor_alternate_name"`
}

func NewOUI(oui string, vendorName string, vendorAlternateName string) (m OUIData, e error) {
	_, err := strconv.ParseInt(oui, 16, 48)
	if err != nil {
		log.Fatal(err)
		return m, errors.New("Unable to parse OUI")
	}

	m.OUI = oui
	m.VendorName = strings.TrimSpace(vendorName)
	m.VendorAlternateName = strings.TrimSpace(vendorAlternateName)
	return m, nil
}

func ListOUIs() []*OUIData {
	return allOUIs
}

func GetOUI(oui string) *OUIData {
	for _, o := range allOUIs {
		if o.OUI == strings.ToUpper(oui) {
			return o
		}
	}
	return nil
}

func makeMACHashMap(fileName string) map[string]*OUIData {
	log.Println("Attempting to load OUIs and build hash map")

	data, err := os.ReadFile(fileName)
	if err != nil {
		panic(err)
	}

	text := string(data)
	// Compile regex expression to match interesting lines
	ouiExp := regexp.MustCompile(`(?m)^([0-9a-fA-F]{2}(-[0-9a-fA-F]{2}){2})\s+\(hex\)\s+(?P<vendorName>.*)\n(?P<OUI>[0-9a-fA-F]{6})\s+\(base 16\)\s+(?P<vendorAlternateName>.*)$`)

	matches := ouiExp.FindAllStringSubmatch(text, -1)
	results := map[string]*OUIData{}

	for _, match := range matches {
		oui := match[ouiExp.SubexpIndex("OUI")]
		vendor := match[ouiExp.SubexpIndex("vendorName")]
		vendorOtherName := match[ouiExp.SubexpIndex("vendorAlternateName")]

		ouiEntry, err := NewOUI(oui, vendor, vendorOtherName)
		if err != nil {
			continue
		}
		results[ouiEntry.OUI] = &ouiEntry
		allOUIs = append(allOUIs, &ouiEntry) // Messy but might as well append instead of opening the file again in another function
	}

	log.Printf("Finished loading OUI hash map, %v OUIs loaded", len(allOUIs))
	return results
}

func normalizeMac(mac string) (string, error) {
	r := regexp.MustCompile("[^a-zA-Z0-9 ]+")
	mac = r.ReplaceAllString(mac, "")

	if len(mac) != 12 {
		return "", errors.New("Incorrect format of MAC address")
	}

	return mac, nil
}

func GetVendorFromMAC(mac string) string {
	// Normalize MAC and then check first 6 characters
	macToFind, err := normalizeMac(mac)
	if err != nil {
		log.Printf("Unable to find MAC %v", mac)
		return ""
	}
	macOUI := GetOUI(macToFind[:6])
	if macOUI != nil {
		return macOUI.VendorName
	}

	return ""
}

type MACHandler struct {
}

func (m MACHandler) GetOUIFromMAC(w http.ResponseWriter, r *http.Request) {
	macParam := chi.URLParam(r, "mac")
	vendor  := GetVendorFromMAC(macParam)
	if vendor == "" {
		http.Error(w, "Unable to find OUI Vendor for this MAC address", http.StatusNotFound)
	}
	err := json.NewEncoder(w).Encode(vendor)
	if err != nil {
		http.Error(w, "Internal Error", http.StatusInternalServerError)
		return
	}
}

type OUIHandler struct {
}

func (m OUIHandler) ListOUIS(w http.ResponseWriter, r *http.Request) {
	err := json.NewEncoder(w).Encode(ListOUIs())
	if err != nil {
		http.Error(w, "Internal Error", http.StatusInternalServerError)
		return
	}
}

func (m OUIHandler) GetOUI(w http.ResponseWriter, r *http.Request) {
	ouiParam := chi.URLParam(r, "oui")
	oui := GetOUI(ouiParam)
	if oui == nil {
		http.Error(w, "OUI not found", http.StatusNotFound)
	}
	err := json.NewEncoder(w).Encode(oui)
	if err != nil {
		http.Error(w, "Internal Error", http.StatusInternalServerError)
		return
	}
}

func ouiRoutes() chi.Router {
	r := chi.NewRouter()
	ouiHandler := OUIHandler{}
	r.Get("/", ouiHandler.ListOUIS)
	r.Get("/{oui}", ouiHandler.GetOUI)
	return r
}

func macRoutes() chi.Router {
	r := chi.NewRouter()
	macHandler := MACHandler{}
	r.Get("/{mac}", macHandler.GetOUIFromMAC)
	return r
}

func main() {
	// Load MAC data
	makeMACHashMap("oui.txt")

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("app is ok!"))
	})
	r.Mount("/oui", ouiRoutes())
	r.Mount("/mac", macRoutes())
	log.Println("Starting web server...")
	http.ListenAndServe(":3000", r)
}
