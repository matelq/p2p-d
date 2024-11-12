const canvas = document.getElementById('gameCanvas');
const context = canvas.getContext('2d');
canvas.width = window.innerWidth;
canvas.height = window.innerHeight;

// Параметры карты
const mapWidth = 3000;
const mapHeight = 3000;

// Камера
let cameraX = 0;
let cameraY = 0;

// Скорость камеры
const cameraSpeedFactor = 1.8; 

// Класс игрока
class Player {
    constructor(id, x, y, size) {
        this.id = id;
        this.x = x;
        this.y = y;
        this.size = size;
        this.baseSpeed = 5;
    }

    get speed() {
        return this.baseSpeed / Math.sqrt(this.size);
    }

    moveTo(x, y) {
        // Игрок движется к координатам мыши
        const dx = x - (canvas.width / 2); // Координаты относительно центра экрана
        const dy = y - (canvas.height / 2);
        const angle = Math.atan2(dy, dx);

        this.x += Math.cos(angle) * this.speed;
        this.y += Math.sin(angle) * this.speed;

        // Ограничим движение игрока границами карты
        this.x = Math.max(this.size, Math.min(this.x, mapWidth - this.size));
        this.y = Math.max(this.size, Math.min(this.y, mapHeight - this.size));
    }
}

// Класс еды
class Food {
    constructor(x, y, size) {
        this.x = x;
        this.y = y;
        this.size = size;
    }
}


let foods = [];
for (let i = 0; i < 100; i++) {
    foods.push(new Food(Math.random() * mapWidth, Math.random() * mapHeight, 5));
}

let currentPlayer = new Player('myId', mapWidth / 2, mapHeight / 2, 10);
let allPlayers = []; // Здесь будут храниться другие игроки



// Отрисовка игрока с учётом смещения камеры
function drawPlayer() {
    context.beginPath();
    context.arc(currentPlayer.x - cameraX, currentPlayer.y - cameraY, currentPlayer.size, 0, Math.PI * 2);
    context.fillStyle = 'blue';
    context.fill();
    context.closePath();
}

// Отрисовка еды с учётом смещения камеры
function drawFoods() {
    foods.forEach(food => {
        context.beginPath();
        context.arc(food.x - cameraX, food.y - cameraY, food.size, 0, Math.PI * 2);
        context.fillStyle = 'green';
        context.fill();
        context.closePath();
    });
}

// Проверка столкновений с едой
function checkCollision(player, food) {
    let distance = Math.sqrt((player.x - food.x) ** 2 + (player.y - food.y) ** 2);
    return distance < player.size + food.size;
}


let foodCounter = 0;

// Функция "съедания" еды
function eatFood() {
    let foodEaten = false; // Переменная для отслеживания, была ли съедена еда

    foods.forEach((food, index) => {
        if (checkCollision(currentPlayer, food)) {
            foodCounter++; // Увеличиваем счетчик
            currentPlayer.size += food.size / 2; // Увеличение размера игрока
            foods.splice(index, 1); // Удаление съеденной еды

            // Спавн новой еды
            foods.push(new Food(Math.random() * mapWidth, Math.random() * mapHeight, 5));
            foodEaten = true; // Обновляем переменную, если еда была съедена
        }
    });

    // Обновляем счетчик только если еда была съедена
    if (foodEaten) {
        updateFoodCounter();
    }
}


function updateFoodCounter(){
    const counterElement = document.getElementById('food-counter');
    counterElement.innerText = `Съедено: ${foodCounter}`;
}

// Отрисовка других игроков
function drawPlayers() {
    allPlayers.forEach(player => {
        context.beginPath();
        context.arc(player.x - cameraX, player.y - cameraY, player.size, 0, Math.PI * 2);
        context.fillStyle = 'red'; // Цвет других игроков
        context.fill();
        context.closePath();
    });
}

// Переменные для хранения координат мыши
let mouseX = canvas.width / 2; // Начальные координаты мыши
let mouseY = canvas.height / 2;

// Обработчик движения мыши
window.addEventListener('mousemove', (event) => {
    mouseX = event.clientX;
    mouseY = event.clientY;
});

// Основной игровой цикл
function gameLoop() {
    context.clearRect(0, 0, canvas.width, canvas.height); // Очистка холста

    // Устанавливаем новое положение камеры
    const targetCameraX = currentPlayer.x - canvas.width / 2;
    const targetCameraY = currentPlayer.y - canvas.height / 2;

    // Увеличиваем скорость камеры
    cameraX += (targetCameraX - cameraX) * cameraSpeedFactor;
    cameraY += (targetCameraY - cameraY) * cameraSpeedFactor;

    // Двигаем игрока к текущим координатам мыши
    currentPlayer.moveTo(mouseX, mouseY);

    // Обработка еды и игрока
    eatFood();
    
    // Отрисовка объектов
    drawFoods();
    drawPlayer();
    drawPlayers();

    requestAnimationFrame(gameLoop);
}


// Установка WebSocket
const socket = new WebSocket('ws://localhost:8080');

socket.onopen = function() {
    console.log('Подключено к серверу');
    socket.send('Привет от клиента!');
};

socket.onmessage = function(event) {
    console.log('Сообщение от сервера:', event.data);
    // Здесь обрабатывай сообщения от сервера (например, обновления других игроков)
};

// Начало игрового цикла
gameLoop();
