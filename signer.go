package main

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

const (
	threads = 6
)

func ExecutePipeline(jobs ...job) {
	var wg sync.WaitGroup
	input := make(chan interface{})

	for _, function := range jobs {
		wg.Add(1)
		output := make(chan interface{})
		go startWorkers(&wg, function, input, output)
		input = output
	}
	wg.Wait()
}

func startWorkers(wg *sync.WaitGroup, function job, in, out chan interface{}) {
	defer wg.Done()
	defer close(out)
	function(in, out)
}

func SingleHash(input, output chan interface{}) {
	var wg sync.WaitGroup

	for i := range input {
		data := fmt.Sprintf("%v", i)
		md5hash := DataSignerMd5(data)
		wg.Add(1)

		go WorkerSingleHash(&wg, data, md5hash, output)
	}
	wg.Wait()
}

func WorkerSingleHash(wg *sync.WaitGroup, data, md5hash string, output chan interface{}) {
	defer wg.Done()

	crcMd5Chan := make(chan string)
	crc32Chan := make(chan string)

	getHash := func(ch chan string, index string, f func(string) string) {
		result := f(index)
		ch <- result
	}

	go getHash(crcMd5Chan, md5hash, DataSignerCrc32)
	go getHash(crc32Chan, data, DataSignerCrc32)

	crc32Md5Hash := <-crcMd5Chan
	crc32Hash := <-crc32Chan

	output <- crc32Hash + "~" + crc32Md5Hash
}

func MultiHash(in, output chan interface{}) {
	var wg sync.WaitGroup

	for i := range in {
		wg.Add(1)
		go WorkerMultiHash(&wg, i, output)
	}

	wg.Wait()
}

func GetMultiHash(wg *sync.WaitGroup, str string, results []string, index int) {
	defer wg.Done()

	crc32hash := DataSignerCrc32(str)
	results[index] = crc32hash
}

func WorkerMultiHash(wg *sync.WaitGroup, value interface{}, output chan interface{}) {
	var workerWG sync.WaitGroup
	results := make([]string, threads)

	defer wg.Done()

	for i := 0; i < threads; i++ {
		workerWG.Add(1)
		data := fmt.Sprintf("%v%v", i, value)
		go GetMultiHash(&workerWG, data, results, i)
	}
	workerWG.Wait()
	multiHash := strings.Join(results, "")

	output <- multiHash
}

func CombineResults(in, out chan interface{}) {
	var results []string

	for i := range in {
		results = append(results, i.(string))
	}

	sort.Strings(results)
	sortedResults := strings.Join(results, "_")
	out <- sortedResults
}
