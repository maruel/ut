ut (utiltest)
=============

Collection of small functions to shorten Go test cases.

Requires Go 1.2 due to the use of `testing.TB`. If needed, replace with
`*testing.T` at the cost of not being usable in benchmarks.


Examples
--------

    func TestFoo(t* testing.T) {
        ut.AssertEqual(Foo(), 42)
    }

    func TestItoa(t* testing.T) {
        data := []struct {
            in       int
            expected string
        }{
            {9, "9"},
            {11, "10"},
        }
        for i, item := range data {
            AssertEqualIndex(t, i, item.expected, strconv.Itoa(item.in))
        }
    }
