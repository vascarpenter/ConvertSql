# ConvertSql

- MySQL (TablePlus) からダンプされた SQLファイルを Oracle フォーマットに変換するCLIプログラム
- 自分のデータで試しているだけで、無理矢理変換なのですべてに当てはまると思わない方がいい
  - varcharはNVARCHAR2
  - floatはFLOAT(126)
  - intはNUMBERに無理矢理変換
- NOT NULLは対応した
- primary key indexは指定できないので、自分でSQLを書いてやること
- oracleでのprimary key作成例：
```
  CREATE UNIQUE INDEX "PK1" ON "sample" ("ID");
  ALTER TABLE "sample" ADD CONSTRAINT "PK1" PRIMARY KEY ("ID")
  USING INDEX "PK1"  ENABLE;
```

- ビルド
```
go build .
mv m ConvertSQL
```

- 使い方
```
./ConvertSQL <filename> [<flags>]
-h,--help  help
```

- 使用例：
```
./ConvertSQL sample_mysql.sql > convert.sql
```
