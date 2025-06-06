// Code generated by Warden. DO NOT EDIT.

package _example

import (
	"fmt"
	warden "github.com/egsam98/warden"
	another "github.com/egsam98/warden/_example/another"
	"net/url"
	"regexp"
	"slices"
	"strconv"
)

var regexData_tqoli = regexp.MustCompile("(.).,(.*)$")
var regexData_swrok = regexp.MustCompile("(.).,(.*)$")

func (self *Data2) Validate() error {
	var errs warden.Errors
	if self.A == "" {
		self.A = "allo da"
	}
	return errs.AsError()
}

func (self *Data) Validate() error {
	var errs warden.Errors
	if !regexData_tqoli.MatchString(self.A.String()) {
		errs.Add("a", warden.Error(fmt.Sprintf("must match regex %s", "(.).,(.*)$")))
	}
	if self.B == nil {
		errs.Add("b", warden.Error("required"))
	}
	if self.B != nil {
		if err := validateB(*self.B); err != nil {
			errs.Add("b", err)
		}
	}
	if self.B != nil {
		if !slices.Contains([]int{another.Allo, 2, 3}, *self.B) {
			errs.Add("b", warden.Error(fmt.Sprintf("must be one of %v", []int{another.Allo, 2, 3})))
		}
	}
	if self.C == "" {
		errs.Add("c", warden.Error("required"))
	}
	if _, err := url.Parse(self.C); err != nil {
		errs.Add("c", warden.Error("must be URL"))
	}
	if !slices.Contains([]string{another.One, "two", "three"}, self.C) {
		errs.Add("c", warden.Error(fmt.Sprintf("must be one of %v", []string{another.One, "two", "three"})))
	}
	if len(self.Arr) < another.Allo {
		errs.Add("arr", warden.Error(fmt.Sprintf("must have length %v min", another.Allo)))
	}
	if len(self.Arr) > 34 {
		errs.Add("arr", warden.Error(fmt.Sprintf("must have length %v max", 34)))
	}
	errs.Add("arr", func() error {
		var errs warden.Errors
		for i, elem := range self.Arr {
			if len(elem) == 0 {
				errs.Add(strconv.Itoa(i), warden.Error("must be non empty"))
			}
			errs.Add(strconv.Itoa(i), func() error {
				var errs warden.Errors
				for i, elem := range elem {
					if !regexData_swrok.MatchString(elem) {
						errs.Add(strconv.Itoa(i), warden.Error(fmt.Sprintf("must match regex %s", "(.).,(.*)$")))
					}
					if len(elem) != another.Allo {
						errs.Add(strconv.Itoa(i), warden.Error(fmt.Sprintf("must have length: %v", another.Allo)))
					}
					if _, err := url.Parse(elem); err != nil {
						errs.Add(strconv.Itoa(i), warden.Error("no url"))
					}
				}
				return errs.AsError()
			}())
		}
		return errs.AsError()
	}())
	errs.Add("arr2", func() error {
		var errs warden.Errors
		for i, elem := range self.Arr2 {
			if elem != nil {
				errs.Add(strconv.Itoa(i), elem.Validate())
			}
		}
		return errs.AsError()
	}())
	if self.Data2 == nil {
		errs.Add("data2", warden.Error("required"))
	}
	if self.Data2 != nil {
		errs.Add("data2", self.Data2.Validate())
	}
	errs.Add("Data3", func() error {
		self := &self.Data3
		var errs warden.Errors
		if self.Test == false {
			errs.Add("test", warden.Error("required"))
		}
		return errs.AsError()
	}())
	if self.Time.IsZero() {
		errs.Add("time", warden.Error("required"))
	}
	if self.Duration == 0 {
		self.Duration = 30000000000 // 30s
	}
	return errs.AsError()
}
