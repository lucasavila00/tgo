package policyresultbad

func bare() (result int) { return }

func captured() (result int) {
    read := func() int { return result }
    _ = read()
    result = 1
    return
}

func switched(value int) (result int) {
    switch value { default: return }
}

func jumped() (result int) {
    goto done
    result = 1
done:
    return
}

var literal = func() (result int) { return }

func typeSwitched(value any) (result int) {
    switch value.(type) { default: return }
}

func looped() (result int) {
    for { return }
}

func loopPost(run bool) (result int) {
    for ; run; _ = result {
        continue
        result = 1
    }
    result = 2
    return
}

func conditionalContinue(run bool) (result int) {
    for ; run; _ = result {
        if run { continue }
        return 1
    }
    result = 2
    return
}

func selected(values <-chan int) (result int) {
    select {
    case <-values: return
    default: return
    }
}

func ranged(values []int) (result int) {
    for range values { return }
    result = 1
    return
}
