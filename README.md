## toy-language
toy language for fun


### lox语法 
使用go实现的lox语言

1.变量
```javascript
var a; //声明变量
print a; // 打印nil

var a, b = 12.4, c = "string"; //声明多个变量
print a; // 打印nil
print b + 3; //打印15.4
print c; //打印 string
```

2.控制语句
```javascript
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
```javascript
var a = [1, 12.4, "string", false, nil];
print a;

a[0] = 2;
print a[0];

for (var i =0; i < len(a); i++) {
  print a[i];
}
```

5..函数
```javascript
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
  }
}
var c = makefn();
c();
c();
```

6.面向对象
```javascript
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
BostonCream().cook();
```