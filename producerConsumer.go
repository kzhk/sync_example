package main

import (
    "fmt"
    "github.com/abiosoft/semaphore"
    "math/rand"
    "runtime"
    "sync"
    "sync/atomic"
    "time"
)

var counter int

func prodConsIncorrect() {
    wg := new(sync.WaitGroup)

    wg.Add(1)
    go func() {
        for i := 0; i < 100000; i++ {
            counter++
        }
        wg.Done()
    }()

    wg.Add(1)
    go func() {
        for i := 0; i < 100000; i++ {
            counter--
        }
        wg.Done()
    }()

    wg.Wait()
}

func prodConsMutex() {
    wg := new(sync.WaitGroup)
    m := new(sync.Mutex)

    wg.Add(1)
    go func() {
        for i := 0; i < 100000; i++ {
            m.Lock()
            counter++
            m.Unlock()
        }
        wg.Done()
    }()

    wg.Add(1)
    go func() {
        for i := 0; i < 100000; i++ {
            m.Lock()
            counter--
            m.Unlock()
        }
        wg.Done()
    }()

    wg.Wait()
}

func prodConsPeterson() {
    var (
        flag [2]int32
        turn int32
    )
    wg := new(sync.WaitGroup)
    n := 1000000

    wg.Add(1)
    go func() {
        for i := 0; i < n; i++ {
            atomic.StoreInt32(&flag[0], 1) // flag[0] = true
            atomic.StoreInt32(&turn, 1)    // turn = 1
            for atomic.LoadInt32(&flag[1]) == 1 && atomic.LoadInt32(&turn) == 1 {
            } // for flag[1] && turn == 1 {}

            counter++
            atomic.StoreInt32(&flag[0], 0) // flag[0] = false
        }
        wg.Done()
    }()

    wg.Add(1)
    go func() {
        for i := 0; i < n; i++ {
            atomic.StoreInt32(&flag[1], 1) // flag[1] = true
            atomic.StoreInt32(&turn, 0)    // turn = 0
            for atomic.LoadInt32(&flag[0]) == 1 && atomic.LoadInt32(&turn) == 0 {
            } // for flag[1] && turn == 0 {}

            counter--
            atomic.StoreInt32(&flag[1], 0) // flag[1] = false
        }
        wg.Done()
    }()

    wg.Wait()
}

func prodConsPeterson2() {
    var (
        flag [2]bool
        turn int
    )
    wg := new(sync.WaitGroup)
    n := 1000000

    wg.Add(1)
    go func() {
        for i := 0; i < n; i++ {
            flag[0] = true
            turn = 1
            for flag[1] && turn == 1 {
            }

            counter++

            flag[0] = false
        }
        wg.Done()
    }()

    wg.Add(1)
    go func() {
        for i := 0; i < n; i++ {
            flag[1] = true
            turn = 0
            for flag[0] && turn == 0 {
            }

            counter--
            flag[1] = false
        }
        wg.Done()
    }()

    wg.Wait()
}

func prodConsSwap() {
    wg := new(sync.WaitGroup)
    n := 1000000
    var lock int32 // Global.
    atomic.StoreInt32(&lock, 0)

    wg.Add(1)
    go func() {
        for i := 0; i < n; i++ {
            for !atomic.CompareAndSwapInt32(&lock, 0, 1) {
            }

            counter++

            atomic.StoreInt32(&lock, 0)
        }
        wg.Done()
    }()

    wg.Add(1)
    go func() {
        for i := 0; i < n; i++ {
            for !atomic.CompareAndSwapInt32(&lock, 0, 1) {
            }

            counter--

            atomic.StoreInt32(&lock, 0)
        }
        wg.Done()
    }()

    wg.Wait()
}

func boundedBufferSemaphore() {
    const bufferSize int = 20
    n := 100
    const producerSpeed int = 1
    const consumerSpeed int = 1 // 10
    in := 0
    out := 0
    var buffer [bufferSize]int
    mutex := semaphore.New(1)
    empty := semaphore.New(bufferSize)
    full := semaphore.New(bufferSize)
    // full.AcquireMany(bufferSize) // Initialized to zero.
    full.DrainPermits()
    wg := new(sync.WaitGroup)

    wg.Add(1)
    go func() {
        for i := 0; i < n; i++ {
            empty.Acquire()
            mutex.Acquire()

            buffer[in] = i
            in = (in + 1) % bufferSize
            fmt.Printf("Produced %d\n", i)

            mutex.Release()
            full.Release()

            time.Sleep(time.Duration(producerSpeed) * time.Millisecond)
        }
        wg.Done()
    }()

    wg.Add(1)
    go func() {
        for i := 0; i < n; i++ {
            full.Acquire()
            mutex.Acquire()

            // Consume buffer[out]
            out = (out + 1) % bufferSize
            fmt.Printf("Consumed %d\n", i)

            mutex.Release()
            empty.Release()
            time.Sleep(time.Duration(consumerSpeed) * time.Millisecond)

        }
        wg.Done()
    }()

    wg.Wait()
}

// Readers-Writers
func rwlockExample() {
    runtime.GOMAXPROCS(runtime.NumCPU())
    // rand.Seed(time.Now().UnixNano())
    // rand.Seed(0)
    l := new(sync.RWMutex)
    wg := new(sync.WaitGroup)
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(i int) {
            r := rand.Intn(10)
            time.Sleep(50 * time.Millisecond)
            if r < 5 {
                // I am reader.
                fmt.Printf("Reader waiting %d\n", i)
                l.RLock()
                fmt.Printf("go reader %d\n", i)
                l.RUnlock()
                wg.Done()
            } else {
                // I am writer
                fmt.Printf("Writer waiting %d\n", i)
                l.Lock()
                fmt.Printf("go writer %d\n", i)
                time.Sleep(50 * time.Millisecond)
                l.Unlock()
                wg.Done()
            }
        }(i)
    }
    wg.Wait()
}

// Dining philosopers
const (
    THINKING = iota
    HUNGRY
    EATING
)

var state [5]int
var self [5]*sync.Cond
var m sync.Mutex

func takeForks(i int) {
    m.Lock()
    defer m.Unlock()
    // fmt.Printf("takeForks: i=%d\n", i)
    state[i] = HUNGRY
    dpTest(i)
    if state[i] != EATING {
        // dpTest(i)
        self[i].Wait()
    }
}

func returnForks(i int) {
    m.Lock()
    defer m.Unlock()
    // fmt.Printf("returnForks: i=%d\n", i)
    state[i] = THINKING
    dpTest((i + 4) % 5)
    dpTest((i + 1) % 5)
}

func dpTest(i int) {
    if (state[(i + 4) % 5] != EATING) &&
    (state[i] == HUNGRY) &&
    (state[(i + 1) % 5] != EATING) {
        state[i] = EATING
        self[i].Signal()
    }
}

func diningPhilosophers() {
    wg := new(sync.WaitGroup)

    for i := 0; i < 5; i++ {
        state[i] = THINKING
        self[i] = sync.NewCond(new(sync.Mutex))
    }

    for i := 0; i < 5; i++ {
        fmt.Printf("diningPhilosphers i=%d\n", i)
        wg.Add(1)
        go func(j int) {
            for k := 0; k < 50; k++ {
                takeForks(j)
                // Eating.
                returnForks(j)
            }
            wg.Done()
        }(i)
    }

    wg.Wait()
}

func main() {
    runtime.GOMAXPROCS(runtime.NumCPU())
    counter = 0
    prodConsIncorrect()
    fmt.Printf("counter=%d\n", counter)

    counter = 0
    prodConsMutex()
    fmt.Printf("counter=%d\n", counter)

    counter = 0
    prodConsPeterson()
    fmt.Printf("counter=%d\n", counter)

    counter = 0
    prodConsPeterson2()
    fmt.Printf("counter=%d\n", counter)

    counter = 0
    prodConsSwap()
    fmt.Printf("counter=%d\n", counter)

    boundedBufferSemaphore()

    rwlockExample()

    diningPhilosophers()

    fmt.Println("main() Finished.")
}
