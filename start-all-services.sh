#!/bin/bash
# Start all services for rhythm-driven error generation demo

echo "🚀 Starting All Services"
echo "=" * 70

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if running on macOS (for terminal commands)
if [[ "$OSTYPE" == "darwin"* ]]; then
    TERM_CMD="open -a Terminal"
else
    TERM_CMD="gnome-terminal --"
fi

echo -e "${YELLOW}This will open 3-4 terminal windows${NC}"
echo ""
echo "Services:"
echo "  1. Slogan Server (port 8080)"
echo "  2. Error Generator (port 9090)"
echo "  3. Demo Script (Terminal)"
echo "  4. [Optional] Fix Generator (port 7070)"
echo ""
read -p "Continue? [y/N]: " confirm

if [ "$confirm" != "y" ]; then
    echo "Cancelled."
    exit 0
fi

# Terminal 1: Slogan Server
echo -e "${GREEN}Starting Slogan Server...${NC}"
osascript -e 'tell application "Terminal" to do script "cd '"$(pwd)"'/slogan-server && go run main.go"'
sleep 2

# Terminal 2: Error Generator
echo -e "${GREEN}Starting Error Generator...${NC}"
osascript -e 'tell application "Terminal" to do script "cd '"$(pwd)"'/error-generator && RHYTHM_SERVICE_URL=http://localhost:5001 go run main.go"'
sleep 2

# Terminal 3: Demo Script
echo -e "${GREEN}Starting Demo Script...${NC}"
osascript -e 'tell application "Terminal" to do script "cd '"$(pwd)"'/rhythm-service && python3 demo_rhythm_errors.py --track \"Where Is My Mind?\" --artist \"Pixies\""'

echo ""
echo -e "${GREEN}✓ All services started${NC}"
echo ""
echo "To add satirical code fixes (optional):"
echo "  1. Terminal 4: cd code-fix-generator && export DEEPSEEK_API_KEY=xxx && python3 satirical_fix_generator.py"
echo "  2. Restart error-generator with: FIX_GENERATOR_URL=http://localhost:7070"
echo ""
echo "Press Ctrl+C in each terminal to stop services"
