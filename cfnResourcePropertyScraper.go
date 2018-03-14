// cfnTemplateResourceScraper.go
// The program scrapes the Cloudformation user guide html pages and outputs a JSON file
// No concurrency

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/sundernswamy/paraparser"
	"io/ioutil"
	"log"
	"strings"
)

const path = "http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/"
const page = "aws-product-property-reference.html"

var parseKeys = []string{"Required:", "Type:"}

type Property struct {
	Name        string
	Description string
	FieldType   string
	Required    string
}

type AwsResource struct {
	ResourcePropertyName string
	Properties           []Property
}

func main() {

	doc, err := goquery.NewDocument(path + page)
	if err != nil {
		log.Fatalf("Cound not find url- %s ", path+page)
	}

	allAWSResources := make([]AwsResource, 0)

	doc.Find(".topictitle").Each(func(i int, s *goquery.Selection) {
		s.Siblings().Each(func(i int, s *goquery.Selection) {
			s.Find("li").Each(func(i int, s *goquery.Selection) {
				href, ok := s.Children().Attr("href")
				if ok == true {
					//			if i == 58 {
					resourceDef, err := scrapeResourceTemplate(href)

					if err != nil {
						log.Println(err)
					}

					allAWSResources = append(allAWSResources, resourceDef)

					//			}
				}
			})
		})
	})

	resourceJsonDef, err := json.MarshalIndent(allAWSResources, "", "\t")
	if err != nil {
		log.Fatalln(err)
	}

	ioutil.WriteFile("resourceproperties.json", resourceJsonDef, 0644)

}

// The function assumes that the html page will use ".variablelist" as the class for
// defining properties and attributes.
// Only properties and return attributes will have this class
// If the HTML contains "Required:" then it is a property else if is an attribute

func scrapeResourceTemplate(docHref string) (res AwsResource, e error) {

	id := "#" + strings.TrimSuffix(docHref, ".html")

	property := make([]Property, 0)

	doc, err := goquery.NewDocument(path + docHref)
	if err != nil {
		return res, err
	}
	doc.Find(id).Each(func(i int, s *goquery.Selection) {
		res.ResourcePropertyName = cleanString(s.Text())
	})

	fmt.Println("Scraping... ", res.ResourcePropertyName)

	// Get properties
	doc.Find(".variablelist").Each(func(i int, s *goquery.Selection) {
		if strings.Contains(s.Text(), "Required:") {
			props := s

			props.Find("dt").Each(func(i int, s *goquery.Selection) {
				property = append(property, Property{s.Text(), "", "", ""})
			})

			props.Find("dd").Each(func(i int, s *goquery.Selection) {
				property[i] = fillProperties(property[i], s)
			})

			res.Properties = property
		}
	})

	if res.ResourcePropertyName == "" {
		return res, errors.New("Could not locate resource name")
	}
	if len(res.Properties) == 0 {
		return res, errors.New("Could not locate properties of resource: " + res.ResourcePropertyName)
	}

	return
}

func fillProperties(prop Property, s *goquery.Selection) Property {

	parsedTxt := paraparser.Parse(s.Text(), parseKeys)

	prop.Description = cleanString(strings.Join(parsedTxt["_rest"], ""))
	prop.Required = cleanString(strings.Join(parsedTxt["Required:"], ""))
	prop.FieldType = cleanString(strings.Join(parsedTxt["Type:"], ""))

	return prop
}

func cleanString(s string) string {

	flag := false
	newByte := make([]byte, 0)
	for _, v := range []byte(s) {
		if v == 10 {
			continue
		}
		if !(string(v) == " " && flag) {
			newByte = append(newByte, v)
		}
		if string(v) == " " {
			flag = true
		} else {
			flag = false
		}
	}
	return strings.TrimSpace(string(newByte))
}
