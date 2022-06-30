package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/360EntSecGroup-Skylar/excelize"
)

func main() {
	integrationPath := "/Users/i.wirananta/go/src/github.com/tokopedia/gqlserver/gql/mitraapp/integrationTest" // change this
	gqlPathQueries := "/Users/i.wirananta/go/src/github.com/tokopedia/gqlserver/gql/mitraapp/queries.go"       // change this
	gqlPathMutation := "/Users/i.wirananta/go/src/github.com/tokopedia/gqlserver/gql/mitraapp/mutations.go"    // change this
	documentName := "./ITSWEEP.xlsx"

	xlsx := excelize.NewFile()
	sheet1Name := "Sheet1"
	xlsx.SetSheetName(xlsx.GetSheetName(1), sheet1Name)

	xlsx.SetCellValue(sheet1Name, "A1", "Endpoint")
	xlsx.SetCellValue(sheet1Name, "B1", "Type")
	xlsx.SetCellValue(sheet1Name, "C1", "Test Case Name")
	xlsx.SetCellValue(sheet1Name, "D1", "File Name")
	xlsx.SetCellValue(sheet1Name, "E1", "Scenario")
	xlsx.SetCellValue(sheet1Name, "F1", "Expected Response")
	xlsx.SetCellValue(sheet1Name, "G1", "Request Param")
	xlsx.SetCellValue(sheet1Name, "H1", "Status")
	xlsx.SetCellValue(sheet1Name, "I1", "Notes")
	xlsx.SetCellValue(sheet1Name, "J1", "PIC")

	gqlQueriesFile, _ := ioutil.ReadFile(gqlPathQueries)
	gqlMutationFile, _ := ioutil.ReadFile(gqlPathMutation)

	err := xlsx.AutoFilter(sheet1Name, "A1", "J1", "")
	if err != nil {
		log.Fatal("ERROR ", err.Error())
	}

	dvRange := excelize.NewDataValidation(true)
	dvRange.Sqref = "H:H"
	dvRange.SetDropList([]string{"Live", "On Progress", "Not Yet", "Pending", "No TestCase", "Not Checked", "Need Fix", "Wont Do", "Endpoint Need Adjustment"})
	xlsx.AddDataValidation(sheet1Name, dvRange)

	mapGqlListQueries := regexQueries(string(gqlQueriesFile))
	mapGqlListMutation := regexQueries(string(gqlMutationFile))

	var total int
	var prevValue string
	err = filepath.Walk(integrationPath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if path[len(path)-4:] == "json" {
				total++

				jsonFile, err := os.Open(path)
				if err != nil {
					fmt.Println(err)
				}
				defer jsonFile.Close()

				byteValue, _ := ioutil.ReadAll(jsonFile)
				var result IntegrationTest
				content, _ := ioutil.ReadFile(path)
				variables := extractValue(string(content), `"variables":`)

				json.Unmarshal([]byte(byteValue), &result)

				endpointName := ""
				for key := range mapGqlListQueries {
					if regexCheckEndpoint(result.Query, key) {
						endpointName = key
						mapGqlListQueries[key] = false
						xlsx.SetCellValue(sheet1Name, fmt.Sprintf("B%d", total+1), "Queries")
						break
					}
				}
				if endpointName == "" {
					for key := range mapGqlListMutation {
						if regexCheckEndpoint(result.Query, key) {
							endpointName = key
							mapGqlListMutation[key] = false
							xlsx.SetCellValue(sheet1Name, fmt.Sprintf("B%d", total+1), "Mutation")
							break
						}
					}
				}
				if endpointName == "" {
					xlsx.SetCellValue(sheet1Name, fmt.Sprintf("B%d", total+1), "-")
					xlsx.SetCellValue(sheet1Name, fmt.Sprintf("I%d", total+1), "Part of chain test case")
					endpointName = "Not in mitraapp"
				}

				xlsx.SetCellValue(sheet1Name, fmt.Sprintf("A%d", total+1), endpointName)
				xlsx.SetCellValue(sheet1Name, fmt.Sprintf("C%d", total+1), result.QueryName)
				xlsx.SetCellValue(sheet1Name, fmt.Sprintf("D%d", total+1), filepath.Base(path))
				xlsx.SetCellValue(sheet1Name, fmt.Sprintf("F%d", total+1), result.Structure[0].ResponseCode)
				xlsx.SetCellValue(sheet1Name, fmt.Sprintf("G%d", total+1), variables)
				xlsx.SetCellValue(sheet1Name, fmt.Sprintf("H%d", total+1), "Live")

				if prevValue == endpointName && prevValue != "Not in mitraapp" {
					xlsx.MergeCell(sheet1Name, "B"+strconv.Itoa(total+1), "B"+strconv.Itoa(total))
					xlsx.MergeCell(sheet1Name, "A"+strconv.Itoa(total+1), "A"+strconv.Itoa(total))
				}

				prevValue = endpointName

			}
			return nil
		})
	if err != nil {
		log.Println(err)
	}
	combinedMap := make(map[string]string)
	for k := range mapGqlListMutation {
		if mapGqlListMutation[k] {
			combinedMap[k] = "Mutation"
		}
	}
	for k := range mapGqlListQueries {
		if mapGqlListQueries[k] {
			combinedMap[k] = "Queries"
		}
	}

	keys := make([]string, 0, len(combinedMap))

	for k := range combinedMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	fmt.Println("Got a total of " + strconv.Itoa(total) + " testcases")
	for _, k := range keys {

		total++
		xlsx.SetCellValue(sheet1Name, fmt.Sprintf("A%d", total+1), k)
		xlsx.SetCellValue(sheet1Name, fmt.Sprintf("B%d", total+1), combinedMap[k])

	}

	fmt.Println("Scanned a total of " + strconv.Itoa(len(mapGqlListQueries)+len(mapGqlListMutation)) + " endpoint")

	err = xlsx.SaveAs(documentName)
	if err != nil {
		fmt.Println(err)
	}
}

type IntegrationTest struct {
	QueryName string      `json:"queryName"`
	Query     string      `json:"query"`
	Structure []Structure `json:"structure"`
}

type Structure struct {
	Env            string                 `json:"env"`
	ResponseCode   int                    `json:"responseCode"`
	ApiParamMap    map[string]interface{} `json:"apiParamMap"`
	Variables      map[string]interface{} `json:"variables"`
	ResponseString map[string]interface{} `json:"responseString"`
}

func regexQueries(body string) map[string]bool {
	apiList := make(map[string]bool)
	r := regexp.MustCompile(`([a-zA-Z0-9_]*)\([a-zA-Z0-9_]*\:*[a-zA-Z0-9_ !,:\[\]]*\) *\:`)
	matches := r.FindAllStringSubmatch(body, -1)
	for _, v := range matches {
		apiList[v[1]] = true
	}
	return apiList
}

func regexCheckEndpoint(body string, key string) bool {
	r := regexp.MustCompile(`[a-zA-Z]*` + key + `[a-zA-Z]*`)
	matches := r.FindAllStringSubmatch(body, -1)
	for _, v := range matches {
		if key == v[0] {
			return true
		}
	}
	return false
}

func extractValue(body string, key string) string {
	start := strings.Index(body, key)
	nullVariables := strings.Contains(body, `"variables": null`)
	var end, openCurlyBracket, closeCurlyBracket int
	if nullVariables {
		return "null"
	}
	if start < 0 {
		return "{}"
	}
	for i := strings.Index(body, key); i < len(body); i++ {
		if string(body[i]) == "{" {
			openCurlyBracket++
		} else if string(body[i]) == "}" {
			closeCurlyBracket++
			if openCurlyBracket == closeCurlyBracket {
				end = i
				break
			}
		}
	}
	return body[start+12 : end+1]
}
