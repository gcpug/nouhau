# 文字列を１文字ずつ加工して新しい文字列を作るSQL

tag:["google-bigquery"]

SQLは宣言的プログラミング言語なので一般的にループを回すような処理は向いていないと考えられています。
ですが`GENERATE_ARRAY`関数を使うことで処理対象を添字付き配列として表現することができれば、
配列を`UNNEST`して一個ずつ処理して最後に`ARRAY_AGG`で添字でソートして結果をまとめることができます。

ここではカナ文字列をローマ字文字列に変換する例を挙げてやり方を紹介します。
カナ文字ローマ字変換のユースケースとしては、キーワードサジェスチョンなど、ブラウザでローマ字入力ごとに一致するキーワードをつど拾いたい時に、あらかじめカナ入力された文字をローマ字として保存しておく時など挙げられます。

具体的なSQLは以下のようになります。

```SQL
SELECT id, ARRAY_TO_STRING(ARRAY_AGG(romaji ORDER BY i), "")
FROM (
 SELECT id,i,
 CASE WHEN kanachar = "ア" THEN "a"
         WHEN kanachar = "イ" THEN "i"
         WHEN kanachar = "ウ" THEN "u"
         WHEN kanachar = "エ" THEN "e"
         WHEN kanachar = "オ" THEN "o"
         WHEN kanachar = "カ" THEN "ka"
         WHEN kanachar = "キ" THEN "ki"
         WHEN kanachar = "ク" THEN "ku"
         WHEN kanachar = "ケ" THEN "ke"
         WHEN kanachar = "コ" THEN "ko"
         WHEN kanachar = "サ" THEN "sa"
         WHEN kanachar = "シ" THEN "si"
         WHEN kanachar = "ス" THEN "su"
         WHEN kanachar = "セ" THEN "se"
         WHEN kanachar = "ソ" THEN "so"
         WHEN kanachar = "タ" THEN "ta"
         WHEN kanachar = "チ" THEN "chi"
         WHEN kanachar = "ツ" THEN "tsu"
         WHEN kanachar = "テ" THEN "te"
         WHEN kanachar = "ト" THEN "to"
         WHEN kanachar = "ナ" THEN "na"
         WHEN kanachar = "ニ" THEN "ni"
         WHEN kanachar = "ヌ" THEN "nu"
         WHEN kanachar = "ネ" THEN "ne"
         WHEN kanachar = "ノ" THEN "no"
         WHEN kanachar = "ハ" THEN "ha"
         WHEN kanachar = "ヒ" THEN "hi"
         WHEN kanachar = "フ" THEN "fu"
         WHEN kanachar = "ヘ" THEN "he"
         WHEN kanachar = "ホ" THEN "ho"
         WHEN kanachar = "マ" THEN "ma"
         WHEN kanachar = "ミ" THEN "mi"
         WHEN kanachar = "ム" THEN "mu"
         WHEN kanachar = "メ" THEN "me"
         WHEN kanachar = "モ" THEN "mo"
         WHEN kanachar = "ヤ" THEN "ya"
         WHEN kanachar = "ユ" THEN "yu"
         WHEN kanachar = "ヨ" THEN "yo"
         WHEN kanachar = "ラ" THEN "ra"
         WHEN kanachar = "リ" THEN "ri"
         WHEN kanachar = "ル" THEN "ru"
         WHEN kanachar = "レ" THEN "re"
         WHEN kanachar = "ロ" THEN "ro"
         WHEN kanachar = "ワ" THEN "wa"
         WHEN kanachar = "ヲ" THEN "wo"
         WHEN kanachar = "ン" THEN "n"
         WHEN kanachar = "ガ" THEN "ga"
         WHEN kanachar = "ギ" THEN "gi"
         WHEN kanachar = "グ" THEN "gu"
         WHEN kanachar = "ゲ" THEN "ge"
         WHEN kanachar = "ゴ" THEN "go"
         WHEN kanachar = "ザ" THEN "za"
         WHEN kanachar = "ジ" THEN "zi"
         WHEN kanachar = "ズ" THEN "zu"
         WHEN kanachar = "ゼ" THEN "ze"
         WHEN kanachar = "ゾ" THEN "zo"
         WHEN kanachar = "ダ" THEN "da"
         WHEN kanachar = "ヂ" THEN "di"
         WHEN kanachar = "ヅ" THEN "du"
         WHEN kanachar = "デ" THEN "de"
         WHEN kanachar = "ド" THEN "do"
         WHEN kanachar = "バ" THEN "ba"
         WHEN kanachar = "ビ" THEN "bi"
         WHEN kanachar = "ブ" THEN "bu"
         WHEN kanachar = "ベ" THEN "be"
         WHEN kanachar = "ボ" THEN "bo"
         WHEN kanachar = "パ" THEN "pa"
         WHEN kanachar = "ピ" THEN "pi"
         WHEN kanachar = "プ" THEN "pu"
         WHEN kanachar = "ペ" THEN "pi"
         WHEN kanachar = "ポ" THEN "po"
         WHEN kanachar = "ティ" THEN "thi"
         WHEN kanachar = "キャ" THEN "kya"
         WHEN kanachar = "キュ" THEN "kyu"
         WHEN kanachar = "キョ" THEN "kyo"
         WHEN kanachar = "シャ" THEN "sha"
         WHEN kanachar = "シュ" THEN "shu"
         WHEN kanachar = "ショ" THEN "sho"
         WHEN kanachar = "ー" THEN "-"
         WHEN kanachar = " " THEN " "
         WHEN kanachar = "　" THEN " "
         ELSE "" END AS romaji
FROM (
SELECT id, i, SUBSTR(Title, i + 1, 1) AS kanachar
FROM (
    SELECT id, Title, GENERATE_ARRAY(0, CHAR_LENGTH(Title)) AS i
    FROM table), UNNEST(i) AS i
)
)
GROUP BY id
```

カナ文字列が`Title`に入っているのでこれを１文字ずつバラしてローマ字に変換して最後に`id`で`GROUP BY`してローマ字文字列として結合します。
以下のように最初に`Title`を１文字ずつ切り出すためにTitleの文字列長の配列を生成して`UNNEST`でバラして添字を使って`SUBSTR`で１文字ずつ取り出します。
(SUBSTR関数で添字をi+1としているのはSUBSTRでは1から開始となるためです)

```SQL
SELECT id, i, SUBSTR(Title, i + 1, 1) AS kanachar
FROM (
    SELECT id, Title, GENERATE_ARRAY(0, CHAR_LENGTH(Title)) AS i
    FROM table), UNNEST(i) AS i
)
```

`UNNEST`でバラしているのは、SQLでは配列の要素を直接処理するのが厳しいため、１行ごとに配列を持っていたのを配列の要素ごとの行として`ネストを解く`ためになります。
次にカナ文字ごとにCASE式で該当するローマ字に変換します。
最後に文字ごとに変換された結果を文字列として`ARRAY_AGG`を使って結合します。

```SQL
SELECT id, ARRAY_TO_STRING(ARRAY_AGG(romaji ORDER BY i), "")
FROM ...
GROUP BY id
```

ARRAY_AGGは集約関数で第一引数で指定したカラムの集合を配列に変換します。
ARRAY_AGGにはORDERBYを指定することもできるので添字を指定して元の順序を復元できます。
(2019/09/10時点ではSpannerではORDERBYは未対応ですがそれ以外は同じSQLを動かせます。順序もたまたまかもしれませんが手元のクエリでは保たれていました。)


