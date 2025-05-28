case "$1" in
    "start")
        echo "Starting development environment..."
        docker-compose up -d mysql redis
        echo "Database and Redis are running!"
        echo "MySQL: localhost:3306"
        echo "Redis: localhost:6379"
        ;;
    "stop")
        echo "Stopping all services..."
        docker-compose down
        ;;
    "restart")
        echo "Restarting services..."
        docker-compose restart
        ;;
    "logs")
        if [ -z "$2" ]; then
            docker-compose logs -f
        else
            docker-compose logs -f "$2"
        fi
        ;;
    "build")
        echo "Building application..."
        docker-compose build app
        ;;
    "clean")
        echo "Cleaning up Docker resources..."
        docker-compose down -v
        docker system prune -f
        ;;
    *)
        echo "Usage: $0 {start|stop|restart|logs|build|clean}"
        echo "  start  - Start MySQL and Redis"
        echo "  stop   - Stop all services"
        echo "  restart- Restart all services"
        echo "  logs   - Show logs (optionally specify service)"
        echo "  build  - Build the Go application"
        echo "  clean  - Clean up all Docker resources"
        ;;
esac