---
title: "Design a Parking Lot"
url: "https://x.com/Harry_The_Nerd/status/2057821348450402614"
category: "LLD"
date: "2026-05-22"
description: "Classic LLD problem: designing a parking lot system."
lang: "zh-CN"
---

# 设计一个停车场

> 经典的详细设计题目：设计一个停车场系统。
>
> 原文：[https://x.com/Harry_The_Nerd/status/2057821348450402614](https://x.com/Harry_The_Nerd/status/2057821348450402614) · 2026-05-22

![封面图](https://pbs.twimg.com/media/HI7Yk9_aYAAfTvZ.jpg)

## 概述（问题描述）

**为一个停车场设计类和它们之间的关系，要求：**

- 有多个楼层
- 支持不同的车辆类型（摩托车、汽车、卡车）
- 能查询某个车位是否空闲
- 能把车位分配给车辆，也能释放车位

## 实体与类

**1\. Vehicle（基类 + 子类型）**

与其写几个毫不相干的独立类，我们用继承。所有车辆共享一些通用属性，但类型会影响车位的分配。

```java
public enum VehicleType {
    BIKE, CAR, TRUCK
}

public abstract class Vehicle {
    private String licensePlate;
    private VehicleType type;

    public Vehicle(String licensePlate, VehicleType type) {
        this.licensePlate = licensePlate;
        this.type = type;
    }

    public VehicleType getType() { return type; }
    public String getLicensePlate() { return licensePlate; }
}

public class Car extends Vehicle {
    public Car(String licensePlate) {
        super(licensePlate, VehicleType.CAR);
    }
}

public class Bike extends Vehicle {
    public Bike(String licensePlate) {
        super(licensePlate, VehicleType.BIKE);
    }
}

public class Truck extends Vehicle {
    public Truck(String licensePlate) {
        super(licensePlate, VehicleType.TRUCK);
    }
}
```

**2\. Spot 类**

一个车位知道自己的 ID、支持哪种车型，以及是什么时候被占用的。

```java
public class Spot implements Comparable<Spot> {
    private String spotId;
    private VehicleType supportedType;
    private Vehicle currentVehicle;
    private LocalDateTime entryTime;
    private int floorNumber;
    private double pricePerHour;

    public Spot(String spotId, VehicleType supportedType, int floorNumber, double pricePerHour) {
        this.spotId = spotId;
        this.supportedType = supportedType;
        this.floorNumber = floorNumber;
        this.pricePerHour = pricePerHour;
    }

    public boolean isAvailable() {
        return currentVehicle == null;
    }

    public void assignVehicle(Vehicle v) {
        this.currentVehicle = v;
        this.entryTime = LocalDateTime.now();
    }

    public void removeVehicle() {
        this.currentVehicle = null;
        this.entryTime = null;
    }

    public LocalDateTime getEntryTime() { return entryTime; }
    public Vehicle getCurrentVehicle() { return currentVehicle; }
    public String getSpotId() { return spotId; }
    public int getFloorNumber() { return floorNumber; }
    public double getPricePerHour() { return pricePerHour; }
    public VehicleType getSupportedType() { return supportedType; }

    @Override
    public int compareTo(Spot other) {
        return Integer.compare(this.floorNumber, other.floorNumber);
    }
}
```

**3\. Floor 类**

一个楼层就是拥有一批车位，仅此而已。它不负责管理车位可用状态，那是 ParkingLot 的活。

```java
public class Floor {
    private int floorNumber;
    private List<Spot> spots;

    public Floor(int floorNumber) {
        this.floorNumber = floorNumber;
        this.spots = new ArrayList<>();
    }

    public void addSpot(Spot spot) {
        spots.add(spot);
    }

    public List<Spot> getSpots() { return spots; }
    public int getFloorNumber() { return floorNumber; }
}
```

**4\. ParkingLot 类**

ParkingLot 既是楼层的容器，也负责跟踪车位可用情况。它用到了：

```java
Map<VehicleType, PriorityQueue<Spot>> availableSpots
```

PriorityQueue 按楼层号排序车位，因此总是优先分配最近楼层的车位：取出 O(1)，重新插入 O(log n)。

```java
public class ParkingLot {
    private String name;
    private List<Floor> floors;
    private Map<VehicleType, PriorityQueue<Spot>> availableSpots;

    public ParkingLot(String name) {
        this.name = name;
        this.floors = new ArrayList<>();
        this.availableSpots = new HashMap<>();

        for (VehicleType type : VehicleType.values()) {
            availableSpots.put(type, new PriorityQueue<>());
        }
    }

    public void addFloor(Floor floor) {
        floors.add(floor);
        for (Spot spot : floor.getSpots()) {
            availableSpots.get(spot.getSupportedType()).offer(spot);
        }
    }

    public Spot assignSpot(Vehicle vehicle) {
        PriorityQueue<Spot> queue = availableSpots.get(vehicle.getType());

        if (queue == null || queue.isEmpty()) {
            throw new RuntimeException("No available spot for vehicle type: " + vehicle.getType());
        }

        Spot spot = queue.poll();
        spot.assignVehicle(vehicle);
        return spot;
    }

    public double releaseSpot(Spot spot) {
        LocalDateTime exitTime = LocalDateTime.now();
        double bill = BillingService.calculateBill(spot, exitTime);
        spot.removeVehicle();
        availableSpots.get(spot.getSupportedType()).offer(spot);
        return bill;
    }
}
```

**5\. BillingService 类**

单一职责，也就是只负责算账单。输入是一个车位（它带着入场时间和每小时价格）和一个离场时间。

```java
public class BillingService {

    public static double calculateBill(Spot spot, LocalDateTime exitTime) {
        LocalDateTime entryTime = spot.getEntryTime();
        long minutes = Duration.between(entryTime, exitTime).toMinutes();
        double hours = Math.ceil(minutes / 60.0); 
        return hours * spot.getPricePerHour();
    }
}
```

## 类关系小结

![](https://pbs.twimg.com/media/HI7ZfgFbAAAQa7A.jpg)

**聚合 vs 组合：**

- ParkingLot -\> Floor：**组合**（离开停车场，楼层就不存在了）
- Floor -\> Spot：**组合**
- Spot -\> Vehicle：**聚合**（车辆独立存在，只是暂时停在这里）

这里比较重要的几个设计决策：

**Vehicle 基类**：避免重复代码，类型只是一个枚举字段

**Spot 实现 Comparable**：让 PriorityQueue 天然按楼层号排序

**Map<VehicleType, PriorityQueue<Spot\>\>**：类型查找 O(1)，车位分配 O(log n)，比 O(n) 的线性扫描好得多

**BillingService 独立成类**：单一职责，计费逻辑不该塞进 Spot 或 ParkingLot

**Floor 只做一个「傻」容器**：Floor 不管理可用状态，保持简单专注

以上就是全部内容，各位，干杯！！！也欢迎点赞、评论、转发和分享
