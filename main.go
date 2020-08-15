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
	"os"
	"regexp"
	"strings"
)

// server configuration
const RunWebserver = true
const RunAddress = ":5432" //listen and serve on 0.0.0.0:5432 (for windows "localhost:8080")

// constants for ISG configuration values used in setValue() function
const ValueInnenraumTemperatur = "val16"         // i.i (z.B. 24.1)
const ValueWarmwasser = "val22"                  // i.i (z.B. 50.0)
const ValueHeizenKomfortTemperatur = "val16"     // i.i (z.B. 20.0)
const ValueHeizenEcoTemperatur = "val17"         // i.i (z.B. 16.0)
const ValueHeizenSteigungHeizkurve = "val25"     // i.i (z.B. 0.7)
const ValueWarmwasserKomfortTemperatur = "val22" // i.i (z.B. 50.0)
const ValueWarmwasserEcoTemperatur = "val23"     // i.i (z.B. 50.0)

func main() {
	if RunWebserver {
		r := gin.Default()
		// TODO: List endpoints if base URL is called

		// REST endpoint for statistic values
		r.GET("/stats", func(c *gin.Context) {
			// collect values
			var output map[string]string
			output = make(map[string]string)
			extractStatistics(getISGbaseURL()+"?s=1,0", output) // Anlage
			extractStatistics(getISGbaseURL()+"?s=1,2", output) // Wärmepumpe
			getValues(output)
			c.JSON(200, output)
		})

		// REST endpoint for config value modification
		r.GET("/value/:name", func(c *gin.Context) {
			//TODO: add error handling and more expressive value refs
			//TODO: return current config value of no new parameter is provided
			name := c.Param("name")
			newval := c.Query("new")
			setValue(name, newval)
			c.JSON(200, "OK")
		})

		r.Run(RunAddress)
		fmt.Println("Stiebel Eltron ISG API Server started.")
	}
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
func extractStatistics(endpoint string, output map[string]string) {

	body := fetchContent(endpoint)
	//extract keys and values with regex magic
	regex := "<td class=\"key\">(?P<Key>.*)</td>\\n.*<td class=\"value\">(?P<Value>.*)</td>"
	m := regexp.MustCompile(regex)
	matches := m.FindAllStringSubmatch(body, -1)

	//print keys and values on console
	for _, match := range matches {
		output[match[1]] = match[2]
	}

}

// Saves a config value (see defined constants named ValueX) by POST-ing:
//
// Content-Type: application/x-www-form-urlencoded
// save one config value: data = {"name":"val16","value":"24,0"}
// save multiple config values: data = [{"name":"val16","value":"24,0"},{"name":"val17","value":"24,0"},{"name":"val25","value":"0.70"}]
func setValue(key string, value string) string {

	//post new value
	client := resty.New()
	body := fmt.Sprintf(`[{"name":"%s", "value":"%s"}]`, key, value)
	resp, err := client.R().
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetFormData(map[string]string{
			"data": body,
		}).
		Post(getISGbaseURL() + "save.php")
	if err != nil {
		log.Fatal(err)
	}
	result := resp.String()
	return result

}

// Get current config value (see defined constants named ValueX)
//
// Kühlen: http://192.168.1.126/?s=4,2
// <input id="aval456" value="EIN" readonly="readonly" class="dropdown dropdown_wert" style="width:3.6em">
// Heizen: http://192.168.1.126/?s=4,0
// Warmwasser: http://192.168.1.126/?s=4,1
// jsvalues['16']['id']='val16'; jsvalues['16']['val']='24,0';
func getValues(output map[string]string) {

	// Kühlen status
	kuehlenBody := fetchContent(getISGbaseURL() + "?s=4,2") //Kühlen
	regexKuehlen := "<input id=\"aval456\" value=\"(?P<Value>.*)\" readonly=\"readonly\""
	m1 := regexp.MustCompile(regexKuehlen)
	kuehlenStatus := m1.FindStringSubmatch(kuehlenBody)[1]
	output["KUEHLEN"] = kuehlenStatus

	// Heizen status
	heizenBody := fetchContent(getISGbaseURL() + "?s=4,0") //Heizen
	heizenKomfortTemperatur := extractJSvalue(heizenBody, ValueHeizenKomfortTemperatur)
	heizenEcoTemperatur := extractJSvalue(heizenBody, ValueHeizenEcoTemperatur)
	heizenSteigungHeizkurve := extractJSvalue(heizenBody, ValueHeizenSteigungHeizkurve)
	output["HEIZEN KOMFORTTEMPERATUR"] = heizenKomfortTemperatur
	output["HEIZEN ECOTEMPERATUR"] = heizenEcoTemperatur
	output["HEIZUNG STEIGUNG-HEIZKURVE"] = heizenSteigungHeizkurve

	// Warmwasser status
	wwBody := fetchContent(getISGbaseURL() + "?s=4,1") //Warmwasser
	wwKomfortTemperatur := extractJSvalue(wwBody, ValueWarmwasserKomfortTemperatur)
	wwEcoTemperatur := extractJSvalue(wwBody, ValueWarmwasserEcoTemperatur)
	output["WARMWASSER KOMFORTTEMPERATUR"] = wwKomfortTemperatur
	output["WARMWASSER ECOTEMPERATUR"] = wwEcoTemperatur

	// Overall status
	startBody := fetchContent(getISGbaseURL() + "?s=0") //Start
	innenraumTemperatur := extractJSvalue2(startBody, ValueInnenraumTemperatur)
	warmwasserTemperatur := extractJSvalue2(startBody, ValueWarmwasser)
	regexBetriebsart := "<input class=\"value curpoi\" readonly=\"readonly\" id=\"aval1\" name=\"aval1\" type=\"text\" value=\"(?P<Value>.*)\" style=\"width:255px\""
	m2 := regexp.MustCompile(regexBetriebsart)
	betriebsart := m2.FindStringSubmatch(startBody)[1]
	output["TEMPERATUR INNENRAUM"] = innenraumTemperatur
	output["TEMPERATUR WARMWASSER"] = warmwasserTemperatur
	output["BETRIEBSART"] = betriebsart

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

// Returns the ISG base URL as specified with command-line argument 1
func getISGbaseURL() string {
	if len(os.Args) > 1 {
		return fmt.Sprintf("http://%s/", os.Args[1])
	} else {
		fmt.Println("ERROR: Please provide ISG IP as first command-line argument")
		os.Exit(-1)
	}
	return ""
}
