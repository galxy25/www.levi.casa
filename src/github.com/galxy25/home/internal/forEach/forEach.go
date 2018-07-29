package internal

import ( /* 👏🏾🔙 */ )

// ForEach functions
// lazily return all values of a collection
// iteration stopper, and iteration error(s)(if any)
// to cancel iteration send on the cancel channel
// Trying a more functional golang approach: Make it so!
// https://golang.org/doc/codewalk/functions/
// https://dave.cheney.net/2016/11/13/do-not-fear-first-class-functions
// Damn I miss my en's
// https://ruby-doc.org/core-2.5.0/Enumerable.html
// https://www.youtube.com/watch?v=IDznovDho7w
// and my CPs
// https://www.martinfowler.com/articles/collection-pipeline/
// https://www.youtube.com/watch?v=i28UEoLXVFQ
type ForEach func() (each chan interface{}, cancel chan struct{}, errs chan error)

// Predicate functions evaluate whether
// predicate applies for current subject
// returning predicate-ability,
// error(if any)
type Predicate func(subject interface{}) (applies bool, err error)

// Select selects a new collection of items
// that satisfy the selector by applying
// the selector for each item in a collection
// stopping at and returning the first iteration error(if any)
func Select(forEach ForEach, selector Predicate) (selected chan interface{}, errs chan error) {
	selected = make(chan interface{})
	errs = make(chan error, 2)
	go func() {
		defer close(selected)
		defer close(errs)
		each, cancel, eachErr := forEach()
		defer close(cancel)
		for {
			select {
			case item, more := <-each:
				if !more {
					continue
				}
				doSelect(item, selector, selected, errs)
			case forEachErr := <-eachErr:
				errs <- forEachErr
				// golang's select is non-deterministic
				for item := range each {
					// Ultimately the heart of select is
					// a for loop inside another for loop. 🤔
					doSelect(item, selector, selected, errs)
				}
				return
			}
		}
	}()
	return selected, errs
}

// Detect detects the first item in the
// collection given for which the predicate holds
// returning that item(if any) and err(s)(if any)
func Detect(forEach ForEach, detector Predicate) (detected interface{}, errs chan error) {
	errs = make(chan error, 1)
	each, cancel, eachErr := forEach()
	for item := range each {
		predicate, predicateErr := detector(item)
		if predicateErr != nil {
			errs <- predicateErr
			break
		}
		if !predicate {
			continue
		}
		detected = item
		break
	}
	close(cancel)
	go func() {
		defer close(errs)
		for err := range eachErr {
			errs <- err
		}
	}()
	return detected, errs
}

// doSelect applies selector to given item
// adding item to selected if result true
// and error(if any) to errs
func doSelect(item interface{}, selector Predicate, selected chan interface{}, errs chan error) {
	predicate, predicateErr := selector(item)
	if predicateErr != nil {
		errs <- predicateErr
		return
	}
	if !predicate {
		return
	}
	selected <- item
}
