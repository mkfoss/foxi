#include "xbase/d4all.h"
#include <stdio.h>

int main() {
    CODE4 codeBase;
    DATA4 *data;
    
    code4init(&codeBase);
    codeBase.accessMode = OPEN4DENY_NONE;
    
    data = d4open(&codeBase, "bank");
    if (data) {
        printf("Successfully opened bank.dbf\n");
        printf("Record count: %ld\n", d4recCount(data));
        d4close(data);
    } else {
        printf("Failed to open bank.dbf\n");
    }
    
    code4initUndo(&codeBase);
    return 0;
}