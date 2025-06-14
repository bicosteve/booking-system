# Hotel Booking System

## Overview

This is a hotel booking system built to manage user registrations, reservations, and payments. The system supports role-based access control, distinguishing between admin and ordinary users. Admin users have elevated privileges such as managing bookings and overseeing system operations, while ordinary users can register, make bookings, and manage their reservations.

The system features seamless integration with Stripe for secure online payments. It also incorporates SMS and email services to notify users upon registration and booking confirmations. Additional features include a password reset mechanism to enhance account security.

## Features

- **User Registration and Authentication**
- **Role-Based Access Control (Admin & User)**
- **Stripe Payment Integration**
- **SMS & Email Notifications**
- **Password Reset Functionality**
- **Swagger API Documentation**

## Technologies Used

- **Golang** – Backend logic and routing
- **Kafka** – Event-driven message processing
- **RabbitMQ** – Event-driven message processing
- **Stripe** – Payment processing
- **Redis** – Caching and session management
- **SMS & Email Integration** – Notifications and alerts

## API Documentation

Swagger UI is available at:

```bash
    http://localhost:7001/swagger/index.html
    http://localhost:7002/swagger/index.html
```

### 🟢 Public Routes

| Method | Endpoint             | Description                        |
| ------ | -------------------- | ---------------------------------- |
| POST   | `/api/user/register` | Register a new user                |
| POST   | `/api/user/login`    | Log in an existing user            |
| GET    | `/api/user/rooms`    | Retrieve a list of available rooms |

### 🔒 Private User Routes (Authentication Required)

| Method | Endpoint                          | Description                     |
| ------ | --------------------------------- | ------------------------------- |
| GET    | `/api/user/me`                    | Get user profile                |
| POST   | `/api/user/reset`                 | Request password reset token    |
| POST   | `/api/user/password-reset`        | Reset user password using token |
| POST   | `/api/user/book`                  | Create a new booking            |
| GET    | `/api/user/book/verify/{room_id}` | Verify a room booking           |
| GET    | `/api/user/book/{room_id}`        | Get booking details for a room  |
| GET    | `/api/user/book/all`              | Get all user bookings           |
| PUT    | `/api/user/book/{booking_id}`     | Update a booking                |

### 🔐 Admin Routes (Admin Authentication Required)

| Method | Endpoint                                 | Description               |
| ------ | ---------------------------------------- | ------------------------- |
| POST   | `/api/admin/rooms`                       | Create a new room         |
| PUT    | `/api/admin/rooms/{room_id}`             | Update room details       |
| DELETE | `/api/admin/rooms/{room_id}`             | Delete a room             |
| GET    | `/api/admin/book/all`                    | Retrieve all bookings     |
| DELETE | `/api/admin/book/{booking_id}/{room_id}` | Delete a specific booking |

### Payloads

```bash
    # 1. register --> POST
    baseurl/user/register
    {
        "email":"user@gmail.com",
        "phone_number":"0706961752",
        "is_vendor":"NO",
        "password":"1234",
        "confirm_password":"1234"
    }

    # 2. login --> POST
    baseurl/user/login
    {
        "email":"user@gmail.com",
        "password":"1234",
    }

    # 3. profile --> GET
    baseurl/user/me


    # 4. Generate reset token --> POST
    baseurl/user/reset
    {
        "email":"vendor@gmail.com"
    }

    # 5. Reset password --> POST
    baseurl/user/password-reset?token={token}
    {
        "password":"12345",
        "confirm-password":"12345"
    }

    # 6. Get Rooms --> GET
    baseurl/user/rooms?room_id={number}&status={VACANT/BOOKED}

    # 7. Create Room --> POST
    baseurl/admin/rooms
    {
        "cost":"7000",
        "status":"VACANT"
    }

    # 8. Update Room --> PUT
    baseurl/admin/rooms/{room_id}
    {
        "cost":10000,
        "status":"BOOKED"
    }

    # 9. Delete Room --> DELETE
    baseurl/admin/rooms/{room_id}

    # 10. Create a booking --> POST
    baseurl/user/book
    {
        "days":5,
        "room_id":1,
        "amount":50000
    }

    # 11. Verify booking --> GET
    baseurl/user/book/verify/{room_id}

    # 12. Update Booking --> PUT
    baseurl/user/book/{booking_id}
    {
        "days":5
    }

    # 13. Get one booking --> GET
    baseurl/user/{booking_id}

    # 14. Get all user's booking --> GET
    baseurl/user/book/all

    # 15.  Admin get property bookings --> GET
    baseurl/admin/book/all

    # 16. Admin Delete Booking --> DELETE
    baseurl/admin/book/{room_id}/{booking_id}

```

## Getting Started

1. **Clone the Repository:**

   ```bash
   git clone https://github.com/your-username/hotel-booking-system.git
   cd hotel-booking-system
   ```

2. **Environment Setup:**

- create a .toml file to hold hold the keys and secrets for stripe, kafka, redis,email,sms providers, jwt secret
- Set up mysql tables by running the file on ./files/

3. **Install Dependancies**

```bash
    go mod download

    go mod tidy
```

4. **Run the application**

```bash
    make servers
```

5. **Access the endpoints**

```bash
    http://localhost:7001/swagger/index.html
    http://localhost:7002/swagger/index.html
```

6. **Build the application**

```bash
    go build -o app ./cmd
```

6. **Prod environment**

```bash
    # This application runs on gcp vm with nginx as proxy.
    # Access the api through
    base-url: http:35.242.242.95/api
    user-endpoints: /user
    admin-endpoints: /admin

    # NB: Use the payloads for each respective endpoing
    # 1. register --> POST
    baseurl/user/register
    {
        "email":"user@gmail.com",
        "phone_number":"0706961752",
        "is_vendor":"NO",
        "password":"1234",
        "confirm_password":"1234"
    }

    # 2. login --> POST
    baseurl/user/login
    {
        "email":"user@gmail.com",
        "password":"1234",
    }

    # 3. profile --> GET
    baseurl/user/me


    # 4. Generate reset token --> POST
    baseurl/user/reset
    {
        "email":"vendor@gmail.com"
    }

    # 5. Reset password --> POST
    baseurl/user/password-reset?token={token}
    {
        "password":"12345",
        "confirm-password":"12345"
    }

    # 6. Get Rooms --> GET
    baseurl/user/rooms?room_id={number}&status={VACANT/BOOKED}

    # 7. Create Room --> POST
    baseurl/admin/rooms
    {
        "cost":"7000",
        "status":"VACANT"
    }

    # 8. Update Room --> PUT
    baseurl/admin/rooms/{room_id}
    {
        "cost":10000,
        "status":"BOOKED"
    }

    # 9. Delete Room --> DELETE
    baseurl/admin/rooms/{room_id}

    # 10. Create a booking --> POST
    baseurl/user/book
    {
        "days":5,
        "room_id":1,
        "amount":50000
    }

    # 11. Verify booking --> GET
    baseurl/user/book/verify/{room_id}

    # 12. Update Booking --> PUT
    baseurl/user/book/{booking_id}
    {
        "days":5
    }

    # 13. Get one booking --> GET
    baseurl/user/{booking_id}

    # 14. Get all user's booking --> GET
    baseurl/user/book/all

    # 15.  Admin get property bookings --> GET
    baseurl/admin/book/all

    # 16. Admin Delete Booking --> DELETE
    baseurl/admin/book/{room_id}/{booking_id}


```

8. **Future Plans**

```bash
    - Increase the test coverage
    - Add docker containers for the application
```
