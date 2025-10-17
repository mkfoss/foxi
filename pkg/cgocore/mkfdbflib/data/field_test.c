#include "xbase/d4all.h"
#include <stdio.h>

int main() {
    CODE4 codeBase;
    DATA4 *data;
    FIELD4 *field;
    
    code4init(&codeBase);
    codeBase.accessMode = OPEN4DENY_NONE;
    
    data = d4open(&codeBase, "bank");
    if (data) {
        printf("Successfully opened bank.dbf\n");
        printf("Record count: %ld\n", d4recCount(data));
        
        // Go to first record
        d4top(data);
        
        // List all fields
        printf("\nFields in bank.dbf:\n");
        for(int i = 1; i <= d4numFields(data); i++) {
            field = d4fieldJ(data, i);
            if (field) {
                printf("Field %d: %s (type: %c, len: %d)\n", 
                       i, f4name(field), f4type(field), f4len(field));
            }
        }
        
        // Read first record data
        printf("\nFirst record data:\n");
        for(int i = 1; i <= d4numFields(data); i++) {
            field = d4fieldJ(data, i);
            if (field) {
                printf("%s: '%s'\n", f4name(field), f4str(field));
            }
        }
        
        d4close(data);
    } else {
        printf("Failed to open bank.dbf\n");
    }
    
    code4initUndo(&codeBase);
    return 0;
}