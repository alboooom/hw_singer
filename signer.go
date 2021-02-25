package main

import (
	"fmt"
	"sort"
	"sync"
	"unicode/utf8"
)

func SingleHash(in, out chan interface{}) {
	mu := &sync.Mutex{}
	wg := &sync.WaitGroup{}
	wg2 := &sync.WaitGroup{}
	mu4 := &sync.Mutex{}

	for val := range in {
		wg.Add(1)
		wg2.Add(1)
		inChan := make(chan string, 1)
		outChan := make(chan string, 2)
		val := fmt.Sprintf("%v", val)
		inChan <- val
		go func(inS chan<- string) {
			defer wg.Done()
			mu.Lock()
			inS <- DataSignerMd5(val)
			mu.Unlock()
			close(inS)
		}(inChan)
		go func(outS chan string) {
			defer wg2.Done()
			wg3 := &sync.WaitGroup{}
			for value := range inChan {
				mu4.Lock()
				val:= value
				mu4.Unlock()
				wg3.Add(1)
				go func() {
					outS <- DataSignerCrc32(val)
					wg3.Done()

				}()
			}
			wg3.Wait()
			a := <-outChan
			b := <-outChan
			out <- (a + "~" + b)
		}(outChan)

	}
	wg.Wait()
	wg2.Wait()
}


func MultiHash(in, out chan interface{}) {
	mu := &sync.Mutex{}
	wg := &sync.WaitGroup{}
	for val := range in {
		var workResult = map[int]string{}
		var mapResult = map[int]string{}
		wg.Add(1)
		go multihashWorker(out, val, mu, wg, workResult, mapResult)
	}
	wg.Wait()

}

func multihashWorker(out chan interface{}, inVal interface{}, mu *sync.Mutex, wg *sync.WaitGroup, workResult, mapResult map[int]string) {
	defer wg.Done()

	workCh := make(chan int, 1)
	for i := 0; i < 6; i++ {
		go func(inVal interface{}, number int, numChan chan int) {
			// defer wg.Done()
			sNum := fmt.Sprintf("%v", number)
			inValString := fmt.Sprintf("%v", inVal)
			result := DataSignerCrc32(sNum + inValString)
			mu.Lock()
			workResult[number] = result
			mu.Unlock()
			numChan <- number
		}(inVal, i, workCh)
	}
	for valS := range workCh {
		mu.Lock()
		mapResult[valS] = workResult[valS]
		mu.Unlock()
		if len(mapResult) == 6 {
			var result string
			for i := 0; i < 6; i++ {
				result += mapResult[i]
			}
			out <- result
			break
		}
	}
}
func CombineResults(in, out chan interface{}) {
	var result []string
	var result2 string

	for val := range in {
		sValue := fmt.Sprintf("%v", val)
		result = append(result, sValue)
	}
	sort.Strings(result)

	for _, res := range result {
		result2 += res + "_"
	}
	_, size := utf8.DecodeLastRuneInString(result2)
	result2 = result2[:len(result2)-size]
	out <- result2

}

func ExecutePipeline(hashJobs ...job) {
	wg := &sync.WaitGroup{}
	in := make(chan interface{}, 1)
	out := make(chan interface{}, 10)

	for _, j := range hashJobs {
		wg.Add(1)
		go func(in, out chan interface{}, worker job) {
			defer wg.Done()
			worker(in, out)
			close(out)
		}(in, out, j)
		in = out
		out = make(chan interface{}, 10)

	}
	wg.Wait()
}
// сюда писать код