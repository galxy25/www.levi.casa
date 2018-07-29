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
// the selector for each iterated item in a collection
// stopping and returning iteration and select error(if any of either)
func Select(forEach ForEach, selector Predicate) (selected chan interface{}, errs chan error) {
	selected = make(chan interface{})
	errs = make(chan error, 2)
	done := make(chan struct{}, 2)
	go func() {
		defer close(selected)
		defer close(errs)
		each, cancel, eachErr := forEach()
		defer close(cancel)
		for {
			select {
			case <-done:
				return
			case err := <-eachErr:
				errs <- err
				done <- struct{}{}
			default:
				item, more := <-each
				if !more {
					continue
				}
				err := doSelect(item, selector, selected)
				if err != nil {
					errs <- err
					done <- struct{}{}
				}
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
	defer close(cancel)
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
	cancel <- struct{}{}
	go func() {
		defer close(errs)
		for err := range eachErr {
			errs <- err
		}
	}()
	return detected, errs
}

// doSelect applies selector to given item
// adding item to selected if predicate true
// and error(if any) to errs
// returning error(if any)
func doSelect(item interface{}, selector Predicate, selected chan interface{}) (err error) {
	predicate, predicateErr := selector(item)
	if predicateErr != nil {
		return predicateErr
	}
	if !predicate {
		return predicateErr
	}
	selected <- item
	return predicateErr
}
