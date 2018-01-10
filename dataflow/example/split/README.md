# Dataflow で単純な分岐を作る例

tag:["google-cloud-dataflow","Python"]

[0, 1, 2, ..., 50] の数値が入力されてくるので、 FizzBuzz っぽく3 の倍数、 5 の倍数、 15 の倍数で分岐させて別々のファイルに書き込む例です。

以下のような 3 つのテキストファイルが出力されます。

```
3
6
9
12
18
21
24
27
33
36
39
42
48
```

```
5
10
20
25
35
40
```

```
0
15
30
45
```

## 方法 1: Additional Output

* [Additional outputs](https://beam.apache.org/documentation/programming-guide/#additional-outputs) という複数の `PCollection` を返す機能があるのでそれを利用します。

```python
import apache_beam as beam


def fizzbuzz(element):
    if element % 15 == 0:
        yield beam.pvalue.TaggedOutput("FizzBuzz", element)
    elif element % 3 == 0:
        yield beam.pvalue.TaggedOutput("Fizz", element)
    elif element % 5 == 0:
        yield beam.pvalue.TaggedOutput("Buzz", element)


p = beam.Pipeline()

fizz, buzz, fizzbuzz = (p | "Read" >> beam.Create(range(50))
                          | "FizzBuzz" >> beam.ParDo(fizzbuzz).with_outputs("Fizz", "Buzz", "FizzBuzz"))

(fizz | "Write Fizz" >> beam.io.WriteToText("fizz.txt"))
(buzz | "Write Buzz" >> beam.io.WriteToText("buzz.txt"))
(fizzbuzz | "Write FizzBuzz" >> beam.io.WriteToText("fizzbuzz.txt"))

p.run().wait_until_finish()
```

## 方法 2: 同じ `PCollection` に対して複数回処理

* 剰余の演算が重複して行われるので効率はあまり良くないです
* 分け方が排他的でない場合はたぶんこっちじゃないと難しいです

```python
import apache_beam as beam


p = beam.Pipeline()

input_pc = (p | "Read" >> beam.Create(range(50)))

fizzbuzz = (input_pc | "FizzBuzz" >> beam.Filter(lambda row: row % 15 == 0))
fizz = (input_pc | "Fizz" >> beam.Filter(lambda row: row % 3 == 0 and row % 15 != 0))
buzz = (input_pc | "Buzz" >> beam.Filter(lambda row: row % 5 == 0 and row % 15 != 0))

(fizz | "Write Fizz" >> beam.io.WriteToText("fizz.txt"))
(buzz | "Write Buzz" >> beam.io.WriteToText("buzz.txt"))
(fizzbuzz | "Write FizzBuzz" >> beam.io.WriteToText("fizzbuzz.txt"))

p.run().wait_until_finish()
```
