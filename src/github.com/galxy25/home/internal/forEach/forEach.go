package internal

import ( /* 👏🏾🔙 */
)

type Each struct {
	Item interface{}
	Err  error
}

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
type ForEach func(cancel <-chan struct{}) (forEach chan Each, err error)

// Predicate functions evaluate whether
// predicate applies for current subject
// returning predicate-ability,
// error(if any)
type Predicate func(subject interface{}) (applies bool, err error)

// Select selects a new collection of items
// that satisfy the selector by applying
// selector for each iterated item in a collection
// returning selected and iteration error(if any)
// selection stops after the first selection error
func Select(forEach ForEach, selector Predicate) (selected chan Each, err error) {
	selected = make(chan Each)
	cancel := make(chan struct{}, 1)
	each, err := forEach(cancel)
	if err != nil {
		close(selected)
		return selected, err
	}
	go func() {
		defer close(selected)
		for item := range each {
			if item.Err != nil {
				continue
			}
			predicate, predicateErr := selector(item.Item)
			if !predicate {
				continue
			}
			selected <- Each{
				Item: item.Item,
				Err:  predicateErr}
			if predicateErr != nil {
				cancel <- struct{}{}
				return
			}
		}
	}()
	return selected, err
}

// Detect detects the first item in the
// collection given for which the predicate holds
// returning detected item and error(if any)
func Detect(forEach ForEach, detector Predicate) (detected interface{}, err error) {
	cancel := make(chan struct{}, 1)
	defer close(cancel)
	each, err := forEach(cancel)
	if err != nil {
		return detected, err
	}
	for item := range each {
		if item.Err != nil {
			continue
		}
		predicate, predicateErr := detector(item.Item)
		if predicateErr != nil {
			cancel <- struct{}{}
			if predicate {
				detected = item.Item
			}
			return detected, predicateErr
		}
		if !predicate {
			continue
		}
		detected = item.Item
		cancel <- struct{}{}
		break
	}
	return detected, err
}
