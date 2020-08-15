// Josef Adersberger licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package main

// ******************************************************************
// API Server for Stiebel Eltron ISG ********************************
// ******************************************************************

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
	"log"
	"regexp"
	"strings"
)

// server configuration
const RunAddress = ":5432"                 //listen and serve on 0.0.0.0:5432 (for windows "localhost:8080")
const ISGAddress = "http://192.168.1.126/" //TODO: add as command line parameter
const ISGStartAddress = ISGAddress + "?s=0"
const ISGAnlagenAddress = ISGAddress + "?s=1,0"
const ISGWaermepumpeAddress = ISGAddress + "?s=1,2"
const ISGHeizenAddress = ISGAddress + "?s=4,0"
const ISGWarmwasserAddress = ISGAddress + "?s=4,1"
const ISGKuehlenAddress = ISGAddress + "?s=4,2"
const ISGSaveAdress = ISGAddress + "save.php"
const RunWebserver = false

// constants for ISG configuration values used in setValue() function
const ValueBetriebsart = "val1"                  // <mode-nummer>
const ValueInnenraumTemperatur = "val16"         // i.i (z.B. 24.1)
const ValueWarmwasser = "val22"                  // i.i (z.B. 50.0)
const ValueHeizenKomfortTemperatur = "val16"     // i.i (z.B. 20.0)
const ValueHeizenEcoTemperatur = "val17"         // i.i (z.B. 16.0)
const ValueHeizenSteigungHeizkurve = "val25"     // i.i (z.B. 0.7)
const ValueWarmwasserKomfortTemperatur = "val22" // i.i (z.B. 50.0)
const ValueWarmwasserEcoTemperatur = "val23"     // i.i (z.B. 50.0)

func main() {

	//TODO: Define REST API based on HTTP router
	if RunWebserver {
		r := gin.Default()
		r.GET("/stats", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "Hallo Du!",
			})
		})
		r.Run(RunAddress)
		fmt.Println("Stiebel Eltron ISG API Server started.")
	}
	//extractStatistics(ISGAnlagenAddress)
	//extractStatistics(ISGWaermepumpeAddress)
	getValues()
	//setValue(ValueHeizenKomfortTemperatur, "24.0")

}

//
//Extract all key/value pairs representing statistics from a given ISG web endpoint.
//The information is extracted according the following structure:
//
//<tr class="odd">
//    <td class="key">VORLAUFISTTEMPERATUR WP</td>
//    <td class="value">21,8°C</td>
//</tr>
//
func extractStatistics(endpoint string) {

	body := fetchContent(endpoint)
	//extract keys and values with regex magic
	regex := "<td class=\"key\">(?P<Key>.*)</td>\\n.*<td class=\"value\">(?P<Value>.*)</td>"
	m := regexp.MustCompile(regex)
	matches := m.FindAllStringSubmatch(body, -1)

	//print keys and values on console
	for _, match := range matches {
		fmt.Printf("%s -> %s \n", match[1], match[2])
	}

}

// Saves a config value (see defined constants named ValueX) by POST-ing:
//
// Content-Type: application/x-www-form-urlencoded
// save one config value: data = {"name":"val16","value":"24,0"}
// save multiple config values: data = [{"name":"val16","value":"24,0"},{"name":"val17","value":"24,0"},{"name":"val25","value":"0.70"}]
func setValue(key string, value string) {

	//post new value
	client := resty.New()
	body := fmt.Sprintf(`[{"name":"%s", "value":"%s"}]`, key, value)
	resp, err := client.R().
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetFormData(map[string]string{
			"data": body,
		}).
		Post(ISGSaveAdress)
	if err != nil {
		log.Fatal(err)
	}
	result := resp.String()
	fmt.Println("Value saved with message: " + result)

}

// Get current config value (see defined constants named ValueX)
//
// Kühlen: http://192.168.1.126/?s=4,2
// <input id="aval456" value="EIN" readonly="readonly" class="dropdown dropdown_wert" style="width:3.6em">
// Heizen: http://192.168.1.126/?s=4,0
// Warmwasser: http://192.168.1.126/?s=4,1
// jsvalues['16']['id']='val16'; jsvalues['16']['val']='24,0';
func getValues() {

	// Kühlen status
	kuehlenBody := fetchContent(ISGKuehlenAddress)
	regexKuehlen := "<input id=\"aval456\" value=\"(?P<Value>.*)\" readonly=\"readonly\""
	m1 := regexp.MustCompile(regexKuehlen)
	kuehlenStatus := m1.FindStringSubmatch(kuehlenBody)[1]
	fmt.Println("Kühlen: " + kuehlenStatus)

	// Heizen status
	heizenBody := fetchContent(ISGHeizenAddress)
	heizenKomfortTemperatur := extractJSvalue(heizenBody, ValueHeizenKomfortTemperatur)
	heizenEcoTemperatur := extractJSvalue(heizenBody, ValueHeizenEcoTemperatur)
	heizenSteigungHeizkurve := extractJSvalue(heizenBody, ValueHeizenSteigungHeizkurve)
	fmt.Println("Heizen Komforttemperatur: " + heizenKomfortTemperatur)
	fmt.Println("Heizen Eco-Temperatur: " + heizenEcoTemperatur)
	fmt.Println("Heizen Steigung Heizkurve: " + heizenSteigungHeizkurve)

	// Warmwasser status
	wwBody := fetchContent(ISGWarmwasserAddress)
	wwKomfortTemperatur := extractJSvalue(wwBody, ValueWarmwasserKomfortTemperatur)
	wwEcoTemperatur := extractJSvalue(wwBody, ValueWarmwasserEcoTemperatur)
	fmt.Println("Warmwasser Komforttemperatur: " + wwKomfortTemperatur)
	fmt.Println("Warmwasser Eco-Temperatur: " + wwEcoTemperatur)

	// Overall status
	startBody := fetchContent(ISGStartAddress)
	innenraumTemperatur := extractJSvalue2(startBody, ValueInnenraumTemperatur)
	warmwasserTemperatur := extractJSvalue2(startBody, ValueWarmwasser)
	fmt.Println("Temperatur Innenraum: " + innenraumTemperatur)
	fmt.Println("Temperatur Warmwasser: " + warmwasserTemperatur)
	regexBetriebsart := "<input class=\"value curpoi\" readonly=\"readonly\" id=\"aval1\" name=\"aval1\" type=\"text\" value=\"(?P<Value>.*)\" style=\"width:255px\""
	m2 := regexp.MustCompile(regexBetriebsart)
	betriebsart := m2.FindStringSubmatch(startBody)[1]
	fmt.Println("Betriebsart: " + betriebsart)
}

// Fetch website content via GET request.
func fetchContent(url string) string {
	client := resty.New()
	resp, err := client.R().
		Get(url)
	if err != nil {
		log.Fatal(err)
	}
	return resp.String()
}

// Extracts JavaScript values: jsvalues['16']['val']='24,0';
func extractJSvalue(body string, key string) string {
	key = strings.Replace(key, "val", "", -1)
	regex := fmt.Sprintf("jsvalues\\['%s'\\]\\['val'\\]='(?P<Value>.*)'", key)
	m := regexp.MustCompile(regex)
	return m.FindStringSubmatch(body)[1]
}

// Extracts JavaScript values (alternative):
//jsobj['id']='val16info';
//jsobj['val']='24,0';
func extractJSvalue2(body string, key string) string {
	regex := fmt.Sprintf("jsobj\\['id'\\]='%s';\\njsobj\\['val'\\]='(?P<Value>.*)';", key+"info")
	m := regexp.MustCompile(regex)
	return m.FindStringSubmatch(body)[1]
}
