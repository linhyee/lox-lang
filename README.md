## lox-language
lox language inspire from http://craftinginterpreters.com/

使用go实现的lox语言 

添加c语言实现

### lox语法 

1.变量
```java
var a; //声明变量
print a; // 打印nil

var a, b = 12.4, c = "string"; //声明多个变量
print a; // nil
print b;  //12.4
print c; // string
```

2.表达式
```java
print 1 + 2; // 3
print "abc" + "def"; // abcdef
print 3.14 * ( 1 + 2); // 9.42
print true ? "true" : "false" // true
print 1, 2, 3, "abc"; // "abc"

var a=0;
print a++;
print --a;(c版本不支持前缀)
```

3.控制语句
```java
var a = true;
if (a) {
  print "true";
} else {
  print "false";
}

var n = 5;
while (n > 0) { 
  print n;
  n--;
}
```
4.数组
```java
var a = [1, 12.4, "string", false, nil];
print a;

a[0] = 2;
print a[0];

for (var i =0; i < len(a); i++) {
  print a[i];
}
```

5.函数
```java
// 函数定义
fun fib(n) {
  if (n <= 1) return n;
  return fib(n -2) + fib(n -1);
}

for (var i =0; i <20; i++) {
  print fib(i);
}

// 支持闭包
fun makefn() {
  var count = 0;
  return fun() { 
    count++;
    print count;
  };
}
var c = makefn();
c();
c();
```

6.面向对象
```java
//定义类
class Boughnut {
  init(a, b) {
    print "构造函数"
    //定义成员变量
    this.x = a;
    this.y = b;
  }
  cook () {
    print "fry until golen brown";
  }
}
var b = Boughnut(1,2) //实例化
print b.x;
print b.y;
b.cook();

//继承
class BostonCream < Doughnut {
  cook() {
   super.cook();
   print "pipe full or custard and coat with chocolate";
  }
}
BostonCream(3,4).cook();
```
