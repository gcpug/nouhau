# Gradle Plugin throws NullPointerException

tag:["google-app-engine", "Java"]

App Engine Java向けに何らかのプロジェクトを Gradle を使って管理する際に、もしそのプロジェクトに何らかの不備がある場合、適切なエラーメッセージが表示されずに `NullPointerException` のみが報告されることがあります。

ここでは、この状況が発生しうる状況と、その解決策について記します。

## `appengine-web.xml` の配置場所の間違い

`appengine-web.xml` の配置場所を間違えると、 `NullPointerException` が発生します。

### 現象

何らかのプロジェクトを Gradle 管理している際に、 `com.google.cloud.tools.appengine` プラグインを `build.gradle` ファイル内で利用することになります。例えば、以下のようになります。

```groovy
buildscript {
    repositories {
        jcenter()
        mavenCentral()
    }
    dependencies {
        classpath 'com.google.cloud.tools:appengine-gradle-plugin:1.+'
    }
}

repositories {
    jcenter()
    mavenCentral()
}

apply plugin: 'java'
apply plugin: 'war'
apply plugin: 'com.google.cloud.tools.appengine'

dependencies {
    compile 'com.google.appengine:appengine-api-1.0-sdk:+'
    providedCompile 'javax.servlet:javax.servlet-api:3.1.0'
}

appengine {
    deploy {
    }
}

group = 'foo.bar'
version = '1.0-SNAPSHOT'

sourceCompatibility = '1.8'
targetCompatibility = '1.8'
```

App Engine Standard Java 向けの場合、 `appengine-web.xml` ファイルを準備します。そして、 `gradle` コマンドにて任意のタスクを実行すると、以下のように `NullPointerException` が発生することがあります。

```bash
$ gradle tasks
FAILURE: Build failed with an exception.
* What went wrong:
A problem occurred configuring root project 'foo-bar'.
> java.lang.NullPointerException (no error message)
* Try:
Run with --stacktrace option to get the stack trace. Run with --info or --debug option to get more log output. Run with --scan to get full insights.
* Get more help at https://help.gradle.org
BUILD FAILED in 0s
```

### 原因

`NullPointerException` が発生するに至った Stacktrace を表示すると、以下のようになります。

```bash
Caused by: java.lang.NullPointerException
        at com.google.cloud.tools.gradle.appengine.flexible.AppEngineFlexiblePlugin$1.execute(AppEngineFlexiblePlugin.java:86)
        at com.google.cloud.tools.gradle.appengine.flexible.AppEngineFlexiblePlugin$1.execute(AppEngineFlexiblePlugin.java:79)
        at org.gradle.configuration.internal.DefaultListenerBuildOperationDecorator$BuildOperationEmittingAction$1$1.run(DefaultListenerBuildOperationDecorator.java:150)
...
```

発生場所が `AppEngineFlexiblePlugin.java:86` であることから、 `com.google.cloud.tools.appengine` プラグインは App Engine Flexible Environment 向けのビルド処理を行おうとしていることがわかります。

App Engine Standard Java の場合、 `appengine-web.xml` ファイルは `src/main/webapp/WEB-INF` ディレクトリに配置されている必要があります。しかし、上記の `NullPointerException` が発生したプロジェクトでは、 `src/main/webapp` ディレクトリの中に配置されていました。このために、 `com.google.cloud.tools.appengine` プラグインは App Engine Flexible Environment 向けのプロジェクトであると判断したようです。

### 解決策

`gradle` コマンドの実行にて `NullPointerException` が発生した際には、App Engine Standard Java 向けのプロジェクトの場合、 `appengine-web.xml` ファイルの配置場所が `src/main/webapp/WEB-INF` ディレクトリかどうかを確認します。もし間違っていた場合は、このディレクトリに正しく配置し、再度 `gradle` コマンドを実行します。
