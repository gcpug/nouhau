# App Engine Reuqest Logを漁る時に使っているQuery

tag:["google-app-engine"]

## protoPayload.lineを1行にまとめる

[read-protoPayload.line.sql](read-protoPayload.line.sql) はprotoPayload.lineに入ってるArrayを適当にJava Script UDFで漁ったときのもの。
ログインユーザのEmailと集計したい値が別の行に保存されていたため、UDFで適当に読んでまとめてみた。