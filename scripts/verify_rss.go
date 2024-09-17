package main

import (
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	success := true
	err := filepath.Walk("./feed", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && (strings.HasSuffix(path, ".xml") || strings.HasSuffix(path, ".rss")) {
			fmt.Printf("Verifying %s...\n", path)
			if err = verifyRSS(path); err != nil {
				fmt.Printf("Validation failed: %v\n", err)
				success = false
			}
		}
		return nil
	})
	if err != nil {
		fmt.Printf("Error walking the path: %v\n", err)
		os.Exit(1)
	}
	if !success {
		os.Exit(1)
	}
}

func verifyRSS(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading file: %v", err)
	}

	var rss RSS
	err = xml.Unmarshal(data, &rss)
	if err != nil {
		return fmt.Errorf("invalid XML in %s: %v", filePath, err)
	}

	if err = rss.Validate(); err != nil {
		return fmt.Errorf("error in %s: %v", filePath, err)
	}

	fmt.Printf("RSS file verification passed for %s!\nf", filePath)
	return nil
}

type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Channel Channel  `xml:"channel"`
}

type Channel struct {
	Title       string `xml:"title"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	Items       []Item `xml:"item"`
}

type Item struct {
	Title       string `xml:"title"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
}

func (r *RSS) Validate() error {
	rules := []func() error{
		r.validatePubDate,
		r.validateItemGUIDs,
		r.validateItemDates,
		r.validatePubDateUpdated,
	}
	var errs []error
	for _, rule := range rules {
		if err := rule(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (r *RSS) validatePubDate() error {
	err := validateRSSDate(r.Channel.PubDate)
	if err != nil {
		return fmt.Errorf("channel pub date: %v", err)
	}
	return nil
}

func (r *RSS) validateItemDates() error {
	for _, item := range r.Channel.Items {
		err := validateRSSDate(item.PubDate)
		if err != nil {
			return fmt.Errorf("item '%s' pub date: %v", item.Title, err)
		}
	}
	return nil
}

func (r *RSS) validateItemGUIDs() error {
	guids := make(map[string]bool)
	for _, item := range r.Channel.Items {
		if guids[item.GUID] {
			return fmt.Errorf("duplicate GUID found: %s", item.GUID)
		}
		guids[item.GUID] = true
	}
	return nil
}

func (r *RSS) validatePubDateUpdated() error {
	if len(r.Channel.Items) == 0 {
		return nil
	}

	chanDate, err := time.Parse(time.RFC1123Z, r.Channel.PubDate)
	if err != nil {
		return fmt.Errorf("invalid date format in channel '%s'", r.Channel.PubDate)
	}

	latestItem := r.Channel.Items[0]
	itemDate, err := time.Parse(time.RFC1123Z, latestItem.PubDate)
	if err != nil {
		return fmt.Errorf("invalid date format in item '%s'", latestItem.PubDate)
	}

	if !chanDate.Equal(itemDate) {
		return fmt.Errorf("publication dates of channel and item do not match")
	}

	return nil
}

func validateRSSDate(str string) error {
	t, err := time.Parse(time.RFC1123Z, str)
	if err != nil {
		return fmt.Errorf("invalid date format in %s", str)
	}
	// Check that day of week was set correctly, as it is ignored by time.Parse.
	if str != t.Format(time.RFC1123Z) {
		return fmt.Errorf("day of week is not correct: expected %s, got %s", t.Format(time.RFC1123Z), str)
	}
	return nil
}
