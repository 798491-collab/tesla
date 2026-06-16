@rem ⚙️ 极简本地 Gradle 执行代理
@echo off
set DIR=%~dp0
"%JAVA_HOME%\bin\java" -classpath "%DIR%gradle\wrapper\gradle-wrapper.jar" org.gradle.wrapper.GradleWrapperMain %*