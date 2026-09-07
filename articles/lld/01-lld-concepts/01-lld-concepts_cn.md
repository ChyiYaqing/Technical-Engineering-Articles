---
title: "LLD Concepts"
url: "https://x.com/Harry_The_Nerd/status/2055635966568620167"
category: "LLD"
date: "2026-05-16"
description: "Core low-level design concepts for interviews and practice."
lang: "zh-CN"
---

# 详细设计核心概念

> 面向面试与日常练习的详细设计（LLD）核心概念。
>
> 原文：[https://x.com/Harry_The_Nerd/status/2055635966568620167](https://x.com/Harry_The_Nerd/status/2055635966568620167) · 2026-05-16

![封面图](https://pbs.twimg.com/media/HIcOEsyacAA5Z3d.jpg)

## 基础

详细设计（LLD）是一门把系统需求翻译成干净、可维护的代码结构的手艺。在动手回答「设计一个停车场」「设计一个国际象棋游戏」「设计一个网约车系统」这类题目之前，我们得先把基本功掌握牢：面向对象编程、设计原则和设计模式。这篇文章会把这三块都讲一遍。

## 1\. 面向对象编程（OOP）

OOP 是详细设计的骨架。你设计的一切最终都要通过类、对象以及它们之间的关系来表达。

**四大支柱：**

**封装**

把数据和操作这些数据的方法打包进一个单元（类），并限制外部直接访问内部细节。

```java
public class BankAccount {
    private double balance; // hidden from outside

    public BankAccount(double initialBalance) {
        this.balance = initialBalance;
    }

    public void deposit(double amount) {
        if (amount > 0) balance += amount;
    }

    public void withdraw(double amount) {
        if (amount > 0 && amount <= balance) balance -= amount;
    }

    public double getBalance() {
        return balance;
    }
}
```

类外面的任何人都无法直接碰到 balance，只能走受控的接口。这就是封装，它保护了数据的完整性。

**抽象**

抽象意味着隐藏复杂度，只暴露调用方真正需要的东西。在 Java 里，这通过抽象类和接口来实现。

```java
public interface PaymentProcessor {
    void processPayment(double amount);
    boolean refund(double amount);
}

public class StripeProcessor implements PaymentProcessor {
    @Override
    public void processPayment(double amount) {
        // Stripe-specific HTTP calls, auth, retries are all hidden
        System.out.println("Processing $" + amount + " via Stripe");
    }

    @Override
    public boolean refund(double amount) {
        System.out.println("Refunding $" + amount + " via Stripe");
        return true;
    }
}
```

调用方只认识 PaymentProcessor，至于底下是 Stripe、Razorpay 还是 PayPal，它并不关心。

**继承**

子类从父类继承字段和方法，在此基础上扩展或特化行为。

```java
public class Vehicle {
    protected String brand;
    protected int speed;

    public Vehicle(String brand, int speed) {
        this.brand = brand;
        this.speed = speed;
    }

    public void move() {
        System.out.println(brand + " is moving at " + speed + " km/h");
    }
}

public class ElectricCar extends Vehicle {
    private int batteryLevel;

    public ElectricCar(String brand, int speed, int batteryLevel) {
        super(brand, speed);
        this.batteryLevel = batteryLevel;
    }

    public void chargeBattery() {
        System.out.println("Charging " + brand + "... Battery: " + batteryLevel + "%");
    }
}
```

ElectricCar 白拿了 move()，又在上面加了 chargeBattery()。继承很强大，但当两者的关系不是干净的「is-a」时，优先用组合而不是继承。

**多态**

同一个接口，多种形态。具体调哪个方法在运行时决定。

```java
public class Shape {
    public double area() {
        return 0;
    }
}

public class Circle extends Shape {
    private double radius;
    public Circle(double radius) { this.radius = radius; }

    @Override
    public double area() {
        return Math.PI * radius * radius;
    }
}

public class Rectangle extends Shape {
    private double width, height;
    public Rectangle(double width, double height) {
        this.width = width; this.height = height;
    }

    @Override
    public double area() {
        return width * height;
    }
}

// Usage
List<Shape> shapes = List.of(new Circle(5), new Rectangle(4, 6));
for (Shape s : shapes) {
    System.out.println("Area: " + s.area()); // correct method picked at runtime
}
```

循环并不在意手里拿的是哪种 Shape，它只管调 area()，就能拿到正确的结果。

## 2\. SOLID 设计原则

SOLID 是五条原则的集合，它们让代码更容易扩展、更不容易被改坏。好的详细设计和优秀的详细设计，差别往往就在这里。

**S - 单一职责原则**

一个类应该只有一个变化的理由。

```java
// BAD: this class does too many things
public class UserService {
    public void saveUser(User user) { /* DB logic */ }
    public void sendWelcomeEmail(User user) { /* Email logic */ }
    public String generateReport(User user) { /* Report logic */ }
}

// GOOD: each class owns one concern
public class UserRepository {
    public void save(User user) { /* DB logic only */ }
}

public class EmailService {
    public void sendWelcomeEmail(User user) { /* Email only */ }
}

public class UserReportGenerator {
    public String generate(User user) { /* Report only */ }
}
```

改邮件模板时，只有 EmailService 会动，其他地方都不会坏。

**O - 开闭原则**

对扩展开放，对修改关闭。新增行为靠加新代码，而不是改老代码。

```java
public interface DiscountStrategy {
    double apply(double price);
}

public class SeasonalDiscount implements DiscountStrategy {
    public double apply(double price) { return price * 0.9; }
}

public class LoyaltyDiscount implements DiscountStrategy {
    public double apply(double price) { return price * 0.85; }
}

public class PriceCalculator {
    public double finalPrice(double price, DiscountStrategy strategy) {
        return strategy.apply(price);
    }
}
```

哪天要加一个「学生折扣」，你只需新增一个类，完全不用碰 PriceCalculator。

**L - 里氏替换原则**

子类应该能完全替换掉父类，而不破坏程序。

```java
public class Bird {
    public void fly() {
        System.out.println("Flying...");
    }
}

// VIOLATION: Penguin can't fly so substituting Bird with Penguin breaks callers
public class Penguin extends Bird {
    @Override
    public void fly() {
        throw new UnsupportedOperationException("Penguins can't fly!");
    }
}

// Good practice: Redesign the hierarchy
public interface Flyable {
    void fly();
}

public class Sparrow implements Flyable {
    public void fly() { System.out.println("Sparrow flying"); }
}

public class Penguin {
    public void swim() { System.out.println("Penguin swimming"); }
}
```

如果子类破坏了父类的契约，那说明这层继承关系本身就是错的。

**I - 接口隔离原则**

不要强迫一个类去实现它用不上的方法。多个小而专注的接口，好过一个臃肿的大接口。

```java
// BAD: one bloated interface
public interface Worker {
    void work();
    void eat();
    void sleep();
}

// A robot worker shouldn't implement eat() and sleep()
// GOOD: split by capability
public interface Workable { void work(); }
public interface Eatable   { void eat();  }
public interface Sleepable { void sleep(); }

public class HumanWorker implements Workable, Eatable, Sleepable {
    public void work()  { System.out.println("Human working");  }
    public void eat()   { System.out.println("Human eating");   }
    public void sleep() { System.out.println("Human sleeping"); }
}

public class RobotWorker implements Workable {
    public void work() { System.out.println("Robot working"); }
}
```

**D - 依赖倒置原则**

高层模块不应该依赖低层模块，两者都应该依赖抽象。

```java
// BAD: high-level class is tightly coupled to a specific DB
public class OrderService {
    private MySQLDatabase db = new MySQLDatabase(); // concrete dependency
    public void placeOrder(Order order) { db.save(order); }
}

// GOOD: depend on an interface
public interface Database {
    void save(Order order);
}

public class MySQLDatabase implements Database {
    public void save(Order order) { System.out.println("Saving to MySQL"); }
}

public class MongoDatabase implements Database {
    public void save(Order order) { System.out.println("Saving to MongoDB"); }
}

public class OrderService {
    private final Database db;

    public OrderService(Database db) { // injected from outside
        this.db = db;
    }

    public void placeOrder(Order order) { db.save(order); }
}
```

现在你可以把 MySQLDatabase 换成 MongoDatabase，而完全不用改 OrderService。

## 3\. 设计模式

设计模式是针对常见问题的可复用解法。它们不是完整的程序，而是经过验证的模板，帮助开发者高效解决反复出现的问题。设计模式分三大类：下面我们看三类在详细设计面试和真实系统里都会频繁用到的模式。

设计模式主要分为三大类：

创建型设计模式

结构型设计模式

行为型设计模式

## **1\. 创建型设计模式**

创建型模式关注对象的创建机制。

**单例模式**

单例模式保证一个类在整个应用中只存在一个对象。

这个模式常用于：

- 日志系统
- 数据库连接
- 配置管理器
- 缓存系统
```java
class Singleton {

    private static Singleton instance;

    private Singleton() {}

    public static Singleton getInstance() {

        if(instance == null) {
            instance = new Singleton();
        }

        return instance;
    }
}

public class Main {

    public static void main(String[] args) {

        Singleton s1 = Singleton.getInstance();
        Singleton s2 = Singleton.getInstance();

        System.out.println(s1 == s2);
    }
}
```

**工厂模式**

工厂模式在不向客户端暴露创建逻辑的前提下创建对象。客户端不再直接用 'new' 造对象，而是由工厂决定该创建哪个对象。

```java
interface Vehicle {
    void drive();
}

class Car implements Vehicle {

    public void drive() {
        System.out.println("Driving Car");
    }
}

class Bike implements Vehicle {

    public void drive() {
        System.out.println("Driving Bike");
    }
}

class VehicleFactory {

    static Vehicle getVehicle(String type) {

        if(type.equalsIgnoreCase("car")) {
            return new Car();
        }

        return new Bike();
    }
}

public class Main {

    public static void main(String[] args) {

        Vehicle vehicle = VehicleFactory.getVehicle("car");

        vehicle.drive();
    }
}
```

优点

- 松耦合
- 更好的可扩展性
- 更清爽的对象创建方式

**建造者模式**

建造者模式用来一步一步地构建复杂对象。当一个类字段很多时，它尤其好用。

```java
class User {

    private String name;
    private int age;

    private User(UserBuilder builder) {
        this.name = builder.name;
        this.age = builder.age;
    }

    static class UserBuilder {

        private String name;
        private int age;

        UserBuilder setName(String name) {
            this.name = name;
            return this;
        }

        UserBuilder setAge(int age) {
            this.age = age;
            return this;
        }

        User build() {
            return new User(this);
        }
    }
}

public class Main {

    public static void main(String[] args) {

        User user = new User.UserBuilder()
                            .setName("Harry")
                            .setAge(22)
                            .build();
    }
}
```

## 2\. 结构型设计模式

结构型模式处理类和对象如何组织的问题。

**适配器模式**

适配器模式让互不兼容的接口能协同工作。现实中的例子就是充电转接头。

```java
interface AndroidCharger {
    void charge();
}

class IphoneCharger {

    void chargeIphone() {
        System.out.println("Charging iPhone");
    }
}

class ChargerAdapter implements AndroidCharger {

    private IphoneCharger iphoneCharger;

    ChargerAdapter(IphoneCharger iphoneCharger) {
        this.iphoneCharger = iphoneCharger;
    }

    public void charge() {
        iphoneCharger.chargeIphone();
    }
}

public class Main {

    public static void main(String[] args) {

        AndroidCharger charger =
                new ChargerAdapter(new IphoneCharger());

        charger.charge();
    }
}
```

优点

- 提升兼容性
- 鼓励代码复用
- 减少依赖问题

**装饰器模式**

装饰器模式在不修改对象结构的前提下，动态地为它添加新功能。

```java
interface Coffee {
    int cost();
}

class BasicCoffee implements Coffee {

    public int cost() {
        return 100;
    }
}

class MilkDecorator implements Coffee {

    private Coffee coffee;

    MilkDecorator(Coffee coffee) {
        this.coffee = coffee;
    }

    public int cost() {
        return coffee.cost() + 20;
    }
}

public class Main {

    public static void main(String[] args) {

        Coffee coffee =
                new MilkDecorator(new BasicCoffee());

        System.out.println(coffee.cost());
    }
}
```

输出

```
120
```

牛奶这个功能是动态加上去的。

**外观模式**

外观模式为复杂系统提供一个简化的接口。它不把每个子系统都直接暴露出去，而是提供单一的入口。

```java
class CPU {
    void start() {
        System.out.println("CPU Started");
    }
}

class RAM {
    void load() {
        System.out.println("RAM Loaded");
    }
}

class HardDrive {
    void read() {
        System.out.println("Hard Drive Read");
    }
}

class ComputerFacade {

    private CPU cpu = new CPU();
    private RAM ram = new RAM();
    private HardDrive hardDrive = new HardDrive();

    void startComputer() {

        cpu.start();
        ram.load();
        hardDrive.read();
    }
}

public class Main {

    public static void main(String[] args) {

        ComputerFacade computer = new ComputerFacade();

        computer.startComputer();
    }
}
```

## 3\. 行为型设计模式

行为型模式关注对象之间的通信。

**观察者模式**

观察者模式定义了一对多的依赖关系：当一个对象发生变化时，多个对象会自动收到通知。

例子包括：

- YouTube 订阅
- 通知系统
- 事件监听器
```java
import java.util.*;

interface Observer {
    void update(String message);
}

class User implements Observer {

    private String name;

    User(String name) {
        this.name = name;
    }

    public void update(String message) {
        System.out.println(name + " received: " + message);
    }
}

class YoutubeChannel {

    private List<Observer> subscribers = new ArrayList<>();

    void subscribe(Observer observer) {
        subscribers.add(observer);
    }

    void notifyUsers(String video) {

        for(Observer observer : subscribers) {
            observer.update(video);
        }
    }
}

public class Main {

    public static void main(String[] args) {

        YoutubeChannel channel = new YoutubeChannel();

        channel.subscribe(new User("Harry"));
        channel.subscribe(new User("Alex"));

        channel.notifyUsers("New video uploaded");
    }
}
```

**策略模式**

策略模式允许在运行时动态选择行为。

这个模式被大量用于：

- 支付系统
- 排序机制
- 认证提供方
```java
interface PaymentStrategy {
    void pay(int amount);
}

class CardPayment implements PaymentStrategy {

    public void pay(int amount) {
        System.out.println("Paid using Card");
    }
}

class UpiPayment implements PaymentStrategy {

    public void pay(int amount) {
        System.out.println("Paid using UPI");
    }
}

class ShoppingCart {

    private PaymentStrategy strategy;

    ShoppingCart(PaymentStrategy strategy) {
        this.strategy = strategy;
    }

    void checkout(int amount) {
        strategy.pay(amount);
    }
}

public class Main {

    public static void main(String[] args) {

        ShoppingCart cart =
                new ShoppingCart(new UpiPayment());

        cart.checkout(1000);
    }
}
```

**这些东西是怎么串起来的**

OOP 给你砖块，SOLID 给你用好这些砖块的规则，设计模式则给你应对常见场景的成熟图纸。

在本系列的下一篇文章里，我们会把这里的所有内容应用到一个真实的详细设计题上：**从零设计一个停车场系统**，带类图和完整的 Java 代码。

以上就是全部内容，各位，干杯！
