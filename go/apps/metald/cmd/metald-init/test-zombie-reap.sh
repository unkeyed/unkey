#!/bin/bash
# More comprehensive zombie reaping test

echo "=== Testing zombie reaping ==="

# Build
make build

# Create a C program that creates zombies
cat > zombie-creator.c <<'EOF'
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/wait.h>

int main() {
    printf("Zombie creator started, PID: %d\n", getpid());
    
    // Create some child processes that exit immediately
    for (int i = 0; i < 5; i++) {
        pid_t pid = fork();
        if (pid == 0) {
            // Child exits immediately
            printf("Child %d (PID %d) exiting\n", i, getpid());
            exit(0);
        } else if (pid > 0) {
            printf("Created child PID %d\n", pid);
            // Parent doesn't wait - creates zombie
        }
        usleep(100000); // 100ms between forks
    }
    
    // Keep running for a bit
    printf("Parent continuing without reaping...\n");
    sleep(3);
    printf("Zombie creator exiting\n");
    return 0;
}
EOF

# Compile the zombie creator
gcc -o zombie-creator zombie-creator.c

echo -e "\n--- Running with standard shell (zombies expected) ---"
./zombie-creator &
CREATOR_PID=$!
sleep 1

# Check zombies
echo "Checking for zombies..."
ps aux | grep defunct | grep -v grep || echo "No zombies found"

# Clean up
wait $CREATOR_PID 2>/dev/null

echo -e "\n--- Running with metald-init (zombies should be reaped) ---"
./metald-init -- ./zombie-creator &
INIT_PID=$!
sleep 2

# Check zombies
echo "Checking for zombies..."
ZOMBIE_COUNT=$(ps aux | grep defunct | grep -v grep | wc -l)
echo "Zombie count: $ZOMBIE_COUNT"

# Let it finish
sleep 2
wait $INIT_PID 2>/dev/null

# Clean up
rm -f zombie-creator zombie-creator.c

echo "=== Zombie reaping test completed ==="