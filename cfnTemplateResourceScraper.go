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

const page = "aws-template-resource-type-ref.html"

var parseKeys = []string{"Required:", "Type:", "Update requires:"}

type Property struct {
	Name        string
	Description string
	FieldType   string
	Required    string
	Updates     string
}

type Attribute struct {
	AttributeName string
	AttributeDesc string
}

type AwsResource struct {
	ResourceName string
	Properties   []Property
	Json         string
	Yaml         string
	Doc          string
	Description  string
	ReturnValues []Attribute
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
					// if i == 58 {
					resourceDef, err := scrapeResourceTemplate(href)
					if err != nil {
						log.Println(err)
					}
					allAWSResources = append(allAWSResources, resourceDef)
					// }
				}
			})
		})
	})

	resourceJsonDef, err := json.MarshalIndent(allAWSResources, "", "\t")
	if err != nil {
		log.Fatalln(err)
	}

	ioutil.WriteFile("templateresources.json", resourceJsonDef, 0644)
}

// The function assumes that the html page will use ".variablelist" as the class for
// defining properties and attributes.
// Only properties and return attributes will have this class
// If the HTML contains "Required:" then it is a property else if is an attribute
func scrapeResourceTemplate(docHref string) (res AwsResource, e error) {

	id := "#" + strings.TrimSuffix(docHref, ".html")

	property := make([]Property, 0)
	var json = []string{}
	var yaml = []string{}

	doc, err := goquery.NewDocument(path + docHref)
	if err != nil {
		return res, err
	}

	res.Doc = path + docHref

	doc.Find(id).Each(func(i int, s *goquery.Selection) {
		res.ResourceName = cleanString(s.Text())
	})

	res.Description = doc.Find(id).First().Next().Text()

	fmt.Println("Scraping... ", res.ResourceName)

	// Get Json & YAML

	doc.Find("#JSON").Each(func(i int, s *goquery.Selection) {
		section := s
		section.Find(".programlisting").Each(func(i int, s *goquery.Selection) {
			json = append(json, s.Text())
		})
	})

	doc.Find("#YAML").Each(func(i int, s *goquery.Selection) {
		section := s
		section.Find(".programlisting").Each(func(i int, s *goquery.Selection) {
			yaml = append(yaml, s.Text())
		})
	})

	if len(json) > 0 {
		res.Json = json[0]
	}
	if len(yaml) > 0 {
		res.Yaml = yaml[0]
	}

	// Get properties
	doc.Find(".variablelist").Each(func(i int, s *goquery.Selection) {
		if strings.Contains(s.Text(), "Required:") {
			props := s

			props.Find("dt").Each(func(i int, s *goquery.Selection) {
				property = append(property, Property{s.Text(), "", "", "", ""})
			})

			props.Find("dd").Each(func(i int, s *goquery.Selection) {
				property[i] = fillProperties(property[i], s)
			})

			res.Properties = property
		}
	})

	// Get attributes
	doc.Find(".variablelist").Each(func(i int, s *goquery.Selection) {
		if !(strings.Contains(s.Text(), "Required:")) {
			attrs := s

			attributes := make([]Attribute, 0)

			attrs.Find("dt").Each(func(i int, s *goquery.Selection) {
				attributes = append(attributes, Attribute{s.Text(), ""})
			})

			attrs.Find("dd").Each(func(i int, s *goquery.Selection) {
				attributes[i].AttributeDesc = cleanString(s.Text())
			})

			res.ReturnValues = attributes
		}
	})

	if res.ResourceName == "" {
		return res, errors.New("Could not locate resource name")
	}
	if len(res.Properties) == 0 {
		return res, errors.New("Could not locate properties of resource: " + res.ResourceName)
	}

	return
}

func fillProperties(prop Property, s *goquery.Selection) Property {

	parsedTxt := paraparser.Parse(s.Text(), parseKeys)

	prop.Description = cleanString(strings.Join(parsedTxt["_rest"], ""))
	prop.Required = cleanString(strings.Join(parsedTxt["Required:"], ""))
	prop.FieldType = cleanString(strings.Join(parsedTxt["Type:"], ""))
	prop.Updates = cleanString(strings.Join(parsedTxt["Update requires:"], ""))

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
